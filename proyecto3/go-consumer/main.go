package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"math/rand"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type report struct {
	Country string `json:"country"`
	Planes  int    `json:"warplanes_in_air"`
	Ships   int    `json:"warships_in_water"`
	Time    string `json:"timestamp"`
}

const assigned = "ESP" // carnet termina en 7

// Scripts Lua: actualizan max/min de forma atómica (seguro con varias réplicas).
var maxScript = redis.NewScript(`
local cur = redis.call('GET', KEYS[1])
if (not cur) or (tonumber(ARGV[1]) > tonumber(cur)) then redis.call('SET', KEYS[1], ARGV[1]) end
return 1`)

var minScript = redis.NewScript(`
local cur = redis.call('GET', KEYS[1])
if (not cur) or (tonumber(ARGV[1]) < tonumber(cur)) then redis.call('SET', KEYS[1], ARGV[1]) end
return 1`)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func process(ctx context.Context, rdb *redis.Client, r report) error {
	c := strings.ToUpper(r.Country)
	p := strconv.Itoa(r.Planes)
	s := strconv.Itoa(r.Ships)

	pipe := rdb.Pipeline()
	pipe.Incr(ctx, "count:"+c)
	pipe.ZIncrBy(ctx, "zset:planes", float64(r.Planes), c)
	pipe.ZIncrBy(ctx, "zset:ships", float64(r.Ships), c)
	pipe.HIncrBy(ctx, "freq:planes", p, 1)
	pipe.HIncrBy(ctx, "freq:ships", s, 1)
	maxScript.Eval(ctx, pipe, []string{"max:planes"}, r.Planes)
	minScript.Eval(ctx, pipe, []string{"min:planes"}, r.Planes)
	maxScript.Eval(ctx, pipe, []string{"max:ships"}, r.Ships)
	minScript.Eval(ctx, pipe, []string{"min:ships"}, r.Ships)

	// Serie temporal del país asignado: un stream (XRANGE lo lee Grafana).
	if c == assigned {
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: "ts:" + assigned,
			MaxLen: 5000,
			Approx: true,
			Values: map[string]interface{}{"planes": r.Planes, "ships": r.Ships},
		})
		pipe.Expire(ctx, "ts:"+assigned, 24*time.Hour) // TTL anti-saturación
	}

	if c == assigned {
		minute := time.Now().UTC().Format("15:04")
		pipe.HSet(ctx, "tsmin:planes", minute, r.Planes)
		pipe.HSet(ctx, "tsmin:ships", minute, r.Ships)
		pipe.Expire(ctx, "tsmin:planes", 24*time.Hour)
		pipe.Expire(ctx, "tsmin:ships", 24*time.Hour)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	if rand.Intn(200) == 0 {
		for _, k := range []string{"planes", "ships"} {
			m, err := rdb.HGetAll(ctx, "freq:"+k).Result()
			if err != nil {
				continue
			}
			best, bestN := "", int64(-1)
			for v, n := range m {
				cnt, _ := strconv.ParseInt(n, 10, 64)
				if cnt > bestN {
					best, bestN = v, cnt
				}
			}
			rdb.Set(ctx, "mode:"+k, best, 0)
		}
	}
	return nil
}

func connect(url string) *amqp.Connection {
	for i := 0; i < 60; i++ {
		c, err := amqp.Dial(url)
		if err == nil {
			return c
		}
		log.Println("esperando rabbitmq:", err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("no se pudo conectar a rabbitmq")
	return nil
}

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: env("VALKEY_ADDR", "valkey:6379")})
	for i := 0; i < 60; i++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			break
		}
		log.Println("esperando valkey...")
		time.Sleep(2 * time.Second)
	}

	conn := connect(env("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"))
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	queue := "war_reports"
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		log.Fatal(err)
	}
	ch.Qos(200, 0, false)
	msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Pool de workers (concurrencia): varias goroutines procesan en paralelo.
	workers := 4
	log.Printf("consumer iniciado con %d workers", workers)
	done := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func() {
			for d := range msgs {
				var r report
				if err := json.Unmarshal(d.Body, &r); err != nil {
					d.Nack(false, false) // mensaje inválido: descartar
					continue
				}
				if err := process(ctx, rdb, r); err != nil {
					log.Println("error valkey:", err)
					d.Nack(false, true) // reintentar
					time.Sleep(500 * time.Millisecond)
					continue
				}
				d.Ack(false)
			}
			done <- struct{}{}
		}()
	}
	<-done
}
