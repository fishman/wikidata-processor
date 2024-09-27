package cmd

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/fishman/wikidata-processor/log"
	"github.com/spf13/cobra"
)

var cfgFile string
var language string
var outDir string
var chunkSize int

var rootCmd = &cobra.Command{
	Use:   "wikidata-processor -h",
	Short: "Wikidata RDF processor",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Output directory: %s\n", outDir)
		log.Infof("Chunk size: %d\n", chunkSize)
		processRDF()
	},
}

var englishLangRegex = regexp.MustCompile(`\@([^ ]+) \.$`)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func filterLanguage(input *bufio.Scanner, chunkSize int, wg *sync.WaitGroup, language string) {
	defer wg.Done()

	var chunkCounter int
	var lineCounter int
	var buffer string

	createOutputFile := func(chunkCounter int) (*gzip.Writer, *os.File, error) {
		outputFileName := fmt.Sprintf("%s/output_%d.gz", outDir, chunkCounter)
		outputFile, err := os.Create(outputFileName)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating output file: %v", err)
		}
		gzipWriter := gzip.NewWriter(outputFile)
		return gzipWriter, outputFile, nil
	}

	gzipWriter, outputFile, err := createOutputFile(chunkCounter)
	if err != nil {
		log.Errorf("Error creating initial output file: %v\n", err)
		return
	}
	defer outputFile.Close()
	defer gzipWriter.Close()

	for input.Scan() {
		line := input.Text()

		buffer += line + "\n"

		if line[len(line)-1] == '.' {
			matches := englishLangRegex.FindStringSubmatch(buffer)
			if matches == nil || (len(matches) > 1 && matches[1] == language) {
				_, err := gzipWriter.Write([]byte(buffer))
				if err != nil {
					log.Errorf("Error writing block: %v\n", err)
				}

				lineCounter++

				if (lineCounter % chunkSize) == 0 {
					gzipWriter.Close()
					outputFile.Close()

					chunkCounter++
					lineCounter = 0

					gzipWriter, outputFile, err = createOutputFile(chunkCounter)
					if err != nil {
						log.Errorf("Error creating output file for chunk %d: %v\n", chunkCounter, err)
						return
					}
				}
			}

			buffer = ""
		}
	}

	if err := input.Err(); err != nil {
		log.Errorf("Error reading input: %v\n", err)
	}

	gzipWriter.Close()
	outputFile.Close()
}

func processRDF() {
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		err = os.Mkdir(outDir, 0755)
		if err != nil {
			log.Errorf("Error creating output directory: %v\n", err)
			return
		}
	}

	input := bufio.NewScanner(os.Stdin)

	var wg sync.WaitGroup
	wg.Add(1)

	go filterLanguage(input, chunkSize, &wg, language)

	wg.Wait()

	log.Info("Processing completed.")
}

func init() {
	cobra.OnInitialize()

	rootCmd.Flags().StringVarP(&language, "language", "l", "en", "Select which language should be filtered")
	rootCmd.Flags().StringVarP(&outDir, "output", "o", "output", "Select output directory")
	rootCmd.Flags().IntVarP(&chunkSize, "chunksize", "s", 100000, "Select chunk size for splits")
}
