// Package store maneja la conexion y escritura de datos hacia Valkey.
// Valkey habla el mismo protocolo RESP que Redis, asi que usamos el
// cliente go-redis estandar sin necesidad de una libreria especial.
//
// IMPORTANTE sobre el diseño de claves: el plugin de Grafana para
// Redis/Valkey (redis-datasource) solo soporta un set fijo de
// comandos predefinidos, y NO incluye LRANGE. Por eso el historial
// (RAM y eliminaciones en el tiempo) se guarda como Redis STREAMS
// (XADD), que el plugin SI soporta nativamente (XRANGE/XREVRANGE) y
// mapea directo a columnas de un panel de series de tiempo, sin
// necesidad de parsear JSON. Los rankings usan Sorted Sets (ZADD/
// ZRANGE, tambien soportado), y los valores "actuales" simples usan
// SET/GET para las tarjetas del dashboard.
package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"daemon-pr2-so1/manager"
	"daemon-pr2-so1/parser"
)

const (
	keyRAMCurrentTotal = "pr2so1:ram:total_kb"
	keyRAMCurrentFree  = "pr2so1:ram:free_kb"
	keyRAMCurrentUsed  = "pr2so1:ram:used_kb"
	streamRAM          = "pr2so1:ram:stream"

	keyEliminadosTotal = "pr2so1:eliminados:total"
	streamEliminados   = "pr2so1:eliminados:stream"

	keyRankingRAM = "pr2so1:ranking:ram"
	keyRankingCPU = "pr2so1:ranking:cpu"

	maxStreamEntries = 500 // recorte aproximado para que los streams no crezcan sin limite
)

type Store struct {
	client *redis.Client
	ctx    context.Context
}

// New crea la conexion a Valkey. addr tipicamente sera "localhost:6379"
// cuando el daemon corre en el host y Valkey esta expuesto por el
// docker-compose.
func New(addr string) *Store {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})
	return &Store{client: client, ctx: context.Background()}
}

// Ping verifica que la conexion a Valkey funciona.
func (s *Store) Ping() error {
	pong, err := s.client.Ping(s.ctx).Result()
	if err != nil {
		return fmt.Errorf("no se pudo conectar a Valkey: %w", err)
	}
	if pong != "PONG" {
		return fmt.Errorf("respuesta inesperada de Valkey: %s", pong)
	}
	return nil
}

func (s *Store) Close() error {
	return s.client.Close()
}

// SaveRAMSnapshot registra el estado de RAM de este ciclo: valores
// SET/GET simples para las tarjetas de "Vision General" (dato actual),
// y una entrada en el stream para la grafica de "Evolucion Temporal".
func (s *Store) SaveRAMSnapshot(mem parser.MemInfo) error {
	pipe := s.client.TxPipeline()
	pipe.Set(s.ctx, keyRAMCurrentTotal, mem.TotalKB, 0)
	pipe.Set(s.ctx, keyRAMCurrentFree, mem.FreeKB, 0)
	pipe.Set(s.ctx, keyRAMCurrentUsed, mem.UsedKB, 0)
	pipe.XAdd(s.ctx, &redis.XAddArgs{
		Stream: streamRAM,
		MaxLen: maxStreamEntries,
		Approx: true,
		Values: map[string]interface{}{
			"total_kb": mem.TotalKB,
			"free_kb":  mem.FreeKB,
			"used_kb":  mem.UsedKB,
		},
	})
	_, err := pipe.Exec(s.ctx)
	return err
}

// RecordEliminaciones incrementa el contador acumulado y agrega una
// entrada al stream, para el panel "Contenedores Eliminados a lo largo
// del tiempo" y el contador de eventos eBPF.
func (s *Store) RecordEliminaciones(count int) error {
	if count == 0 {
		return nil
	}
	pipe := s.client.TxPipeline()
	pipe.IncrBy(s.ctx, keyEliminadosTotal, int64(count))
	pipe.XAdd(s.ctx, &redis.XAddArgs{
		Stream: streamEliminados,
		MaxLen: maxStreamEntries,
		Approx: true,
		Values: map[string]interface{}{
			"count": count,
		},
	})
	_, err := pipe.Exec(s.ctx)
	return err
}

// UpdateRankings actualiza los sorted sets de Top RAM/CPU, conservando
// el pico historico de cada contenedor (ZADD GT: solo sube el score,
// nunca baja), para que el ranking siga incluyendo contenedores ya
// eliminados, tal como pide el enunciado.
func (s *Store) UpdateRankings(containers []manager.ContainerMetrics) error {
	for _, cm := range containers {
		member := fmt.Sprintf("%s|%s", cm.Name, cm.ID)

		err := s.client.ZAddArgs(s.ctx, keyRankingRAM, redis.ZAddArgs{
			GT:      true,
			Members: []redis.Z{{Score: float64(cm.RssKB), Member: member}},
		}).Err()
		if err != nil {
			return fmt.Errorf("error actualizando ranking RAM: %w", err)
		}

		err = s.client.ZAddArgs(s.ctx, keyRankingCPU, redis.ZAddArgs{
			GT:      true,
			Members: []redis.Z{{Score: cm.PorcCPU, Member: member}},
		}).Err()
		if err != nil {
			return fmt.Errorf("error actualizando ranking CPU: %w", err)
		}
	}
	return nil
}
