package controllers

import (
	"net/http"

	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

func Health(w http.ResponseWriter, _ *http.Request) {
	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"name":   "agrafa",
	})
}
