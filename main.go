package main

import (
	config "emr-berkas-lab/pkg"
	"io"
	"os"
	"path/filepath"
	"sync"

	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"

	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

var (
	watcher  *fsnotify.Watcher
	mutex    sync.Mutex
	pdfFiles []string
)

func cariBerkasPenunjangLab(rootDir, regpas, trxlab string) ([]string, error) {
	var pdfFiles []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".pdf") {
			if strings.Contains(info.Name(), regpas) && strings.Contains(info.Name(), trxlab) {
				pdfFiles = append(pdfFiles, path)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return pdfFiles, nil
}

func main() {
	config.Load()
	pdfdir := config.Data.Get("dirpdf")

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Basic CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// Kodeoutlet_NoLab_CM_NoKunjungan_NoTransaksiHIS_ThnBlmTglJamCetak
	r.Get("/lab", func(w http.ResponseWriter, r *http.Request) {
		// medrecid := r.URL.Query().Get("medrecid")
		regpas := r.URL.Query().Get("regpas")
		trxlab := r.URL.Query().Get("trxlab")

		pdfFiles, err := cariBerkasPenunjangLab(pdfdir, regpas, trxlab)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(pdfFiles) == 0 {
			http.Error(w, "Tidak ada berkas PDF yang ditemukan dengan parameter yang sesuai.", http.StatusNotFound)
			return
		}

		// Mengambil berkas PDF pertama yang ditemukan
		pdfFile := pdfFiles[0]

		// Membuka berkas PDF
		file, err := os.Open(pdfFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Set header untuk respons HTTP
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(pdfFile)))

		// Salin isi berkas PDF ke respons HTTP
		_, err = io.Copy(w, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	http.ListenAndServe(":9690", r)

	// watchDirectory(pdfdir)

}

// func watchDirectory(dir string) {

// 	// Inisialisasi fsnotify
// 	watcher, err := fsnotify.NewWatcher()
// 	if err != nil {
// 		fmt.Println("Error inisialisasi fsnotify:", err)
// 		return
// 	}
// 	defer watcher.Close()

// 	// Mendaftarkan direktori untuk dipantau
// 	err = watcher.Add(dir)
// 	if err != nil {
// 		fmt.Println("Error menambahkan direktori untuk dipantau:", err)
// 		return
// 	}

// 	// Menjalankan goroutine untuk memantau perubahan pada direktori
// 	go func() {
// 		for {
// 			select {
// 			case event, ok := <-watcher.Events:
// 				if !ok {
// 					return
// 				}
// 				if event.Op&fsnotify.Create == fsnotify.Create {
// 					if strings.HasSuffix(event.Name, ".pdf") {
// 						fmt.Println("Berkas PDF baru ditambahkan:", event.Name)
// 					}
// 				}
// 			case err, ok := <-watcher.Errors:
// 				if !ok {
// 					return
// 				}
// 				fmt.Println("Error fsnotify:", err)
// 			}
// 		}
// 	}()

// 	// Menahan aplikasi agar tetap berjalan
// 	select {}
// }
