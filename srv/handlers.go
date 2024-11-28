package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

// Add these methods to your Server struct in server.go

func (s *Server) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.templates.ExecuteTemplate(w, "index.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *Server) handleViewOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		orderID := vars["id"]

		order, ok := s.orders.Load(orderID)
		if !ok {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}

		data := struct {
			Order *Order
		}{
			Order: order.(*Order),
		}

		err := s.templates.ExecuteTemplate(w, "order.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *Server) handleDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		orderID := vars["id"]

		order, ok := s.orders.Load(orderID)
		if !ok {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}

		o := order.(*Order)
		if o.Status != "complete" {
			http.Error(w, "Adventure generation not complete", http.StatusBadRequest)
			return
		}

		// Set headers for file download
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=adventure-%s.zip", orderID))

		// Serve the file
		http.ServeFile(w, r, o.OutputPath)
	}
}

// Helper function to encode JSON
func encodeJSON(v interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(v)
	return buf
}

// Helper function to create ZIP file
func createZip(sourceDir, zipPath string) error {
	zipfile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's the zip file itself
		if path == zipPath {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Make the path relative to the source directory
		header.Name, err = filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
