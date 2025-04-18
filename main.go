package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	Cameras   []Camera
	OutputDir string
}

type Camera struct {
	IP       string
	Login    string
	Password string
	Name     string
}

func main() {
	mode := flag.String("mode", "capture", "Режим работы: test, capture")
	fps := flag.Int("fps", 1, "Кадров в секунду (1-60)")
	count := flag.Int("count", 10, "Количество кадров в серии")
	interval := flag.Int("interval", 60, "Интервал между сериями (секунды)")
	flag.Parse()

	cfg := Config{
		Cameras: []Camera{
			{"192.168.0.150", "admin", "!Decor_2025", "cam1"},
			{"192.168.0.151", "admin", "!Decor_2025", "cam2"},
		},
		OutputDir: "C:\\CameraPhotos",
	}

	switch *mode {
	case "test":
		testCameras(cfg)
	case "capture":
		startCapture(cfg, *fps, *count, *interval)
	default:
		log.Fatal("Неизвестный режим. Допустимо: test, capture")
	}
}

func testCameras(cfg Config) {
	for _, cam := range cfg.Cameras {
		testCamera(cam)
	}
}

func testCamera(cam Camera) {
	log.Printf("Проверка камеры %s (%s)...", cam.Name, cam.IP)

	cmd := exec.Command("ffmpeg",
		"-rtsp_transport", "tcp",
		"-i", fmt.Sprintf("rtsp://%s:%s@%s/Streaming/Channels/101", cam.Login, cam.Password, cam.IP),
		"-t", "5",
		"-f", "null", "-",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("❌ Ошибка камеры %s: %v", cam.Name, err)
	} else {
		log.Printf("✅ Камера %s доступна", cam.Name)
	}
}

func startCapture(cfg Config, fps, count, interval int) {
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		log.Fatalf("❌ Ошибка создания папки: %v", err)
	}

	for {
		var wg sync.WaitGroup
		for _, cam := range cfg.Cameras {
			wg.Add(1)
			go func(c Camera) {
				defer wg.Done()
				captureSeries(c, fps, count, cfg.OutputDir)
			}(cam)
		}
		wg.Wait()

		if interval > 0 {
			log.Printf("⏳ Следующая серия через %d сек...", interval)
			time.Sleep(time.Duration(interval) * time.Second)
		} else {
			break
		}
	}
}

func captureSeries(cam Camera, fps, count int, outputDir string) {
	timestamp := time.Now().Format("0102_150405")
	rtspURL := fmt.Sprintf("rtsp://%s:%s@%s/Streaming/Channels/101", cam.Login, cam.Password, cam.IP)

	for i := 1; i <= count; i++ {
		filename := fmt.Sprintf("%s_%s_%d-%d.jpg", cam.Name, timestamp, i, count)
		outputPath := filepath.Join(outputDir, filename)

		cmd := exec.Command("ffmpeg",
			"-rtsp_transport", "tcp",
			"-i", rtspURL,
			"-q:v", "1", // Максимальное качество
			"-frames:v", "1", // Один кадр
			"-y", // Перезапись без подтверждения
			outputPath,
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Printf("⚠️ Ошибка %s: %v", cam.Name, err)
		} else {
			log.Printf("✅ %s: сохранён %s", cam.Name, filename)
		}

		if i < count {
			time.Sleep(time.Second / time.Duration(fps))
		}
	}
}
