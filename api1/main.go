package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	nombreAPI = "API1"
	nombreVM  = "VM1"
	carnet    = "201901657"
	puerto    = "8081"
)

// Direcciones de las otras APIs. Ajusta las IPs si tus VMs cambian.
var (
	api2URL = "http://api2:8082"      // API2 vive en la misma VM1
	api3URL = "http://192.168.122.87:8083" // API3 vive en VM2
)

type RespuestaSalud struct {
	Estado      string `json:"status"`
	Mensaje     string `json:"message"`
	MarcaTiempo string `json:"timestamp"`
	VM          string `json:"VM"`
	Carnet      string `json:"carnet"`
}

type RespuestaLlamada struct {
	NombreAPI string `json:"apiname"`
	Mensaje   string `json:"message"`
	Conexion  bool   `json:"connection"`
	Carnet    string `json:"carnet"`
}

func manejadorSalud(respuestaHTTP http.ResponseWriter, solicitud *http.Request) {
	respuesta := RespuestaSalud{
		Estado:      "UP",
		Mensaje:     fmt.Sprintf("%s está lista", nombreAPI),
		MarcaTiempo: time.Now().UTC().Format(time.RFC3339),
		VM:          nombreVM,
		Carnet:      carnet,
	}
	respuestaHTTP.Header().Set("Content-Type", "application/json")
	json.NewEncoder(respuestaHTTP).Encode(respuesta)
}

// manejadorLlamada consulta /health de otra API y genera la respuesta.
func manejadorLlamada(apiDestino, vmDestino, urlBase string) http.HandlerFunc {
	return func(respuestaHTTP http.ResponseWriter, solicitud *http.Request) {
		respuestaHTTP.Header().Set("Content-Type", "application/json")
		cliente := http.Client{Timeout: 3 * time.Second}

		respuesta, errorSolicitud := cliente.Get(urlBase + "/health")
		if errorSolicitud != nil {
			escribirRespuestaLlamada(respuestaHTTP, apiDestino, vmDestino, false)
			return
		}
		defer respuesta.Body.Close()

		cuerpo, _ := io.ReadAll(respuesta.Body)
		var salud RespuestaSalud
		if errorJSON := json.Unmarshal(cuerpo, &salud); errorJSON != nil || salud.Estado != "UP" {
			escribirRespuestaLlamada(respuestaHTTP, apiDestino, vmDestino, false)
			return
		}

		escribirRespuestaLlamada(respuestaHTTP, apiDestino, vmDestino, true)
	}
}

func escribirRespuestaLlamada(respuestaHTTP http.ResponseWriter, apiDestino, vmDestino string, exitoso bool) {
	var mensaje string
	if exitoso {
		mensaje = fmt.Sprintf("La %s ubicada en la %s está funcionando", apiDestino, vmDestino)
	} else {
		mensaje = fmt.Sprintf("ERROR: La %s ubicada en la %s no está funcionando", apiDestino, vmDestino)
	}
	respuesta := RespuestaLlamada{
		NombreAPI: apiDestino,
		Mensaje:   mensaje,
		Conexion:  exitoso,
		Carnet:    carnet,
	}
	json.NewEncoder(respuestaHTTP).Encode(respuesta)
}

func main() {
	enrutador := http.NewServeMux()
	enrutador.HandleFunc("/health", manejadorSalud)
	enrutador.HandleFunc(fmt.Sprintf("/api1/%s/call-api2", carnet), manejadorLlamada("API2", "VM1", api2URL))
	enrutador.HandleFunc(fmt.Sprintf("/api1/%s/call-api3", carnet), manejadorLlamada("API3", "VM2", api3URL))

	log.Printf("%s (VM: %s, carnet: %s) escuchando en el puerto %s...", nombreAPI, nombreVM, carnet, puerto)
	log.Fatal(http.ListenAndServe(":"+puerto, enrutador))
}
