package main

import (
	"fmt"
	"strconv"
	"strings"
)

func worker(workWidth int, out chan [][]byte, up, down chan int){
	workplace := <- out
	start := <- up
	end := <-down
	workHeight := end - start + 1
	temp := make([][]byte, workHeight)
	for i := range workplace {
		temp[i] = make([]byte, workWidth)
	}

	for y := 0; y < workHeight; y++{
		for x := 0; x < workWidth; x++{
			aliveNeighbours := 0
			for i := y - 1; i <= y + 1; i++{
				if i == 0 || i == workHeight {continue}else{
					for j := x - 1; j <= x + 1; j++{
						if i == y || j == x{continue}

						if workplace[i][(j + workWidth) % workWidth] == 255{
							aliveNeighbours += 1
						}
					}
				}
			}
			if workplace[y][x] == 255{
				if aliveNeighbours < 2 || aliveNeighbours > 3{
					temp[y][x] = 0
				}else{temp[y][x] = 255}
			}
			if temp[y][x] == 0{
				if aliveNeighbours == 3{
					temp[y][x] =255
				}else{temp[y][x] = 0}
			}
		}
	}
	out <- temp
}

func buildWorkPlace(p golParams, world [][]byte, indexOfWorker int, up, down chan int)[][]byte{
	numOfThreads := p.threads
	heightOfWorker := p.imageHeight/numOfThreads + 2
	start := indexOfWorker * p.imageHeight/numOfThreads - 1
	end := (indexOfWorker + 1) *p.imageHeight/numOfThreads + 1
	workPlace := make([][]byte, heightOfWorker)
	for i := range workPlace{
		workPlace[i] = make([]byte, p.imageWidth)
	}
	for y := 0; y < p.imageHeight; y++{
		if y < start{continue}
		if y > end{break}else{
			for x := 0; x <= p.imageWidth; x++{
				workPlace[y][x] = world[y][x]
			}
		}
	}
	up <- start
	down <- end
	return workPlace
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

	// Create the 2D slice to store the world.
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}
	// Calculate the new state of Game of Life after the given number of turns.
	for turns := 0; turns < p.turns; turns++ {
		//make a 2-D array to store the information for next round
		nextWorld := make([][]byte, p.imageHeight)
		for i := range nextWorld {
			nextWorld[i] = make([]byte, p.imageWidth)
		}

		out := make(chan [][]byte)
		up := make(chan int)
		down := make(chan int)
		for i := 0; i <= p.threads; i++{
			workPlace := buildWorkPlace(p, world, i, up, down)
			out <- workPlace
		}
		go worker(p.imageWidth, out, up, down)

		start := <- up + 1
		end := <- down + 1
		output :=<- out
		for y := 0; y < p.imageHeight; y++{
			for x := 0; x < p.imageWidth; x++{
				if x < start{continue}
				if x > end{break}else{
					nextWorld[y][x] = output[y][x]
				}
			}
		}


	}

	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
			}
		}
	}

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}
