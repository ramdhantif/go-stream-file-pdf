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
	"github.com/jlaffaye/ftp"

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

func main() {
	config.Load()
	setPort := config.Data.Get("setPort")

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
		pdfdir := config.Data.Get("dirpdf")

		medrecid := r.URL.Query().Get("medrecid")
		regpas := r.URL.Query().Get("regpas")
		trxlab := r.URL.Query().Get("trxlab")
		if regpas == "" || trxlab == "" {
			http.Error(w, "Parameter tidak lengkap.", http.StatusBadRequest)
			return
		}

		pdfFiles, err := cariBerkasPenunjangLab(pdfdir, regpas, medrecid, trxlab)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(pdfFiles) == 0 {
			http.Error(w, "Tidak ada berkas PDF yang ditemukan dengan parameter yang sesuai.", http.StatusNotFound)
			return
		}

		// Mengambil berkas PDF pertama yang ditemukan
		pdfFile := pdfFiles[len(pdfFiles)-1]

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

	r.Get("/ftplab", func(w http.ResponseWriter, r *http.Request) {

		//parameter disini
		medrecid := r.URL.Query().Get("medrecid")
		regpas := r.URL.Query().Get("regpas")
		trxlab := r.URL.Query().Get("trxlab")
		ftp_download := config.Data.Get("ftp_download")

		if regpas == "" || trxlab == "" {
			http.Error(w, "Parameter tidak lengkap.", http.StatusBadRequest)
			return
		}

		pdfFiles, err := cariBerkasPenunjangLab(ftp_download, regpas, medrecid, trxlab)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// jika tidak ada dilokal maka ambil dari ftp server
		if len(pdfFiles) == 0 {
			pdfFiles, err := downloadFileFTP(ftp_download, regpas, medrecid, trxlab)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if len(pdfFiles) == 0 {
				http.Error(w, "Tidak ada berkas PDF yang ditemukan dengan parameter yang sesuai.", http.StatusNotFound)
				return
			}

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

	fmt.Println("web server running, port", setPort)

	http.ListenAndServe(setPort, r)
}

func downloadFileFTP(rootDir, regpas, medrecid, trxlab string) ([]string, error) {
	var pdfFiles []string

	urlftp := config.Data.Get("urlftp")
	ftp_username := config.Data.Get("ftp_username")
	ftp_password := config.Data.Get("ftp_password")
	ftp_dirpdf := config.Data.Get("ftp_dirpdf")
	ftp_download := config.Data.Get("ftp_download")

	conn, err := ftp.Dial(urlftp)
	if err != nil {
		return pdfFiles, err
	}
	defer conn.Quit()

	err = conn.Login(ftp_username, ftp_password)
	if err != nil {
		return pdfFiles, err
	}

	err = conn.ChangeDir(ftp_dirpdf)
	if err != nil {
		return pdfFiles, err
	}

	entries, err := conn.List(".")
	if err != nil {
		return pdfFiles, err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name, ".pdf") {
			if strings.Contains(entry.Name, regpas) && strings.Contains(entry.Name, trxlab) && strings.Contains(entry.Name, medrecid) {

				download, err := conn.Retr(entry.Name)
				if err != nil {
					wg.Done()
					return pdfFiles, err
				}

				dirPenyimpananBerkasLab := ftp_download + "/" + entry.Name
				f, err := os.Create(dirPenyimpananBerkasLab)
				if err != nil {
					wg.Done()
					return pdfFiles, err
				}

				_, err = io.Copy(f, download)
				if err != nil {
					wg.Done()
					return pdfFiles, err
				}

				download.Close()
				f.Close()

				pdfFiles = append(pdfFiles, dirPenyimpananBerkasLab)
			}
		}
	}
	wg.Done()

	wg.Wait()

	return pdfFiles, nil
}

func cariBerkasPenunjangLab(rootDir, regpas, medrecid, trxlab string) ([]string, error) {
	var pdfFiles []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".pdf") {
			if strings.Contains(info.Name(), regpas) && strings.Contains(info.Name(), trxlab) && strings.Contains(info.Name(), medrecid) {
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
