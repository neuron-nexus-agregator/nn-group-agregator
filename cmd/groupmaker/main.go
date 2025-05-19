package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"agregator/group/internal/pkg/app"
)

func main() {
	diff_str := os.Getenv("DIFF")
	diff_int, err := strconv.Atoi(diff_str)

	if err != nil {
		log.Default().Println("Error converting DIFF to int:", err, "using default value")
		diff_int = 85
	}
	if diff_int < 0 || diff_int > 100 {
		diff_int = 85
	}

	diff_float := float64(diff_int) / float64(100)

	alpha_str := os.Getenv("ALPHA")
	alpha_int, err := strconv.Atoi(alpha_str)
	if err != nil {
		log.Default().Println("Error converting ALPHA to int:", err, "using default value")
		alpha_int = 20
	}
	if alpha_int < 0 || alpha_int > 100 {
		alpha_int = 20
	}
	alpha_float := float64(alpha_int) / float64(100)

	distance_str := os.Getenv("DISTANCE")
	distance_int, err := strconv.Atoi(distance_str)
	if err != nil {
		distance_int = 20
	}
	if distance_int < 0 || distance_int > 100 {
		distance_int = 20
	}
	distance_float := float64(distance_int) / float64(100)

	log.Default().Printf("DIFF: %v, ALPHA: %v, DISTANCE: %v\n", diff_float, alpha_float, distance_float)

	app := app.New(diff_float, distance_float, alpha_float, 30*time.Second)
	app.Run()
}
