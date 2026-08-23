package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

const (
	sseRetryDelay   = 5000
	ssePingInterval = 15 * time.Second
)

type sseSnapshotFetcher func(ctx context.Context) (any, error)

func streamSSESnapshots(
	w http.ResponseWriter,
	r *http.Request,
	maxDuration time.Duration,
	interval time.Duration,
	fetch sseSnapshotFetcher,
) {
	initialPayload, err := fetch(r.Context())
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprintf(w, "retry: %d\n\n", sseRetryDelay); err != nil {
		return
	}

	initialBytes, err := marshalSSEPayload(initialPayload)
	if err != nil {
		return
	}

	if err := writeSSEData(w, initialBytes); err != nil {
		return
	}
	flusher.Flush()

	streamDeadline := time.NewTimer(maxDuration)
	defer streamDeadline.Stop()

	snapshotTicker := time.NewTicker(interval)
	defer snapshotTicker.Stop()

	pingTicker := time.NewTicker(ssePingInterval)
	defer pingTicker.Stop()

	lastPayload := initialBytes

	for {
		select {
		case <-r.Context().Done():
			return
		case <-streamDeadline.C:
			return
		case <-pingTicker.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-snapshotTicker.C:
			payload, err := fetch(r.Context())
			if err != nil {
				return
			}

			payloadBytes, err := marshalSSEPayload(payload)
			if err != nil {
				return
			}

			if bytes.Equal(payloadBytes, lastPayload) {
				continue
			}

			if err := writeSSEData(w, payloadBytes); err != nil {
				return
			}
			flusher.Flush()
			lastPayload = payloadBytes
		}
	}
}

type serviceSSEInitialFetcher func(ctx context.Context) (types.ServiceStreamData, error)
type serviceSSEUpdateFetcher func(ctx context.Context, afterID int64) (types.ServiceDetailData, []types.ServiceHistoryEntryData, error)

func streamServiceSSESnapshots(
	w http.ResponseWriter,
	r *http.Request,
	maxDuration time.Duration,
	interval time.Duration,
	fetchInitial serviceSSEInitialFetcher,
	fetchUpdates serviceSSEUpdateFetcher,
) {
	initialPayload, err := fetchInitial(r.Context())
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.WriteError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprintf(w, "retry: %d\n\n", sseRetryDelay); err != nil {
		return
	}

	initialBytes, err := marshalSSEPayload(initialPayload)
	if err != nil || writeSSEData(w, initialBytes) != nil {
		return
	}
	flusher.Flush()

	lastObservationID := int64(0)
	if initialPayload.Observation != nil {
		lastObservationID = initialPayload.Observation.ID
	}
	lastServiceBytes, err := marshalSSEPayload(initialPayload.Service)
	if err != nil {
		return
	}

	streamDeadline := time.NewTimer(maxDuration)
	defer streamDeadline.Stop()

	snapshotTicker := time.NewTicker(interval)
	defer snapshotTicker.Stop()

	pingTicker := time.NewTicker(ssePingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-streamDeadline.C:
			return
		case <-pingTicker.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-snapshotTicker.C:
			service, observations, err := fetchUpdates(r.Context(), lastObservationID)
			if err != nil {
				return
			}

			serviceBytes, err := marshalSSEPayload(service)
			if err != nil {
				return
			}

			emittedObservation := false
			for _, observation := range observations {
				if observation.ID <= lastObservationID {
					continue
				}

				payloadBytes, err := marshalSSEPayload(types.ServiceStreamData{
					Service:     service,
					Observation: &observation,
				})
				if err != nil || writeSSEData(w, payloadBytes) != nil {
					return
				}
				flusher.Flush()
				lastObservationID = observation.ID
				emittedObservation = true
			}

			if emittedObservation {
				lastServiceBytes = serviceBytes
				continue
			}
			if bytes.Equal(serviceBytes, lastServiceBytes) {
				continue
			}

			payloadBytes, err := marshalSSEPayload(types.ServiceStreamData{Service: service})
			if err != nil || writeSSEData(w, payloadBytes) != nil {
				return
			}
			flusher.Flush()
			lastServiceBytes = serviceBytes
		}
	}
}

func marshalSSEPayload(payload any) ([]byte, error) {
	return json.Marshal(payload)
}

func writeSSEData(w http.ResponseWriter, payload []byte) error {
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}

	return nil
}
