package cmd

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fishman/wikidata-processor/log"
	"github.com/knakk/rdf"
	"github.com/spf13/cobra"
)

var cfgFile string
var language string
var outDir string
var chunkSize int

var rootCmd = &cobra.Command{
	Use:   "wikidata-processor [flags] [inputFile|--]",
	Short: "Wikidata RDF processor",
	Args: func(cmd *cobra.Command, args []string) error {
		// Ensure input is either "--" for stdin or a valid file path
		if len(args) == 0 {
			return fmt.Errorf("no input provided; use '--' to read from stdin or provide an input file")
		}
		if args[0] != "--" {
			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				return fmt.Errorf("input file does not exist: %s", args[0])
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var reader io.Reader
		var err error

		if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}

		if args[0] == "--" {
			reader = os.Stdin
		} else {
			reader, err = openFileWithDecompression(args[0])
			if err != nil {
				log.Errorf("Error opening input file: %v\n", err)
				return
			}
		}

		input := bufio.NewScanner(reader)

		var wg sync.WaitGroup
		wg.Add(1)

		go filterLanguage(input, chunkSize, &wg, language)

		wg.Wait()

		log.Info("Processing completed.")
	},
}

func openFileWithDecompression(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".gz":
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("error creating gzip reader: %v", err)
		}
		return gzipReader, nil
	case ".bz2":
		return bzip2.NewReader(file), nil
	default:
		return file, nil
	}
}

var rdfRegex = regexp.MustCompile(`\@([^ ]+) \.$`)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func filterRDF(data []byte, language string) ([]byte, error) {
	var buffer bytes.Buffer
	decoder := rdf.NewTripleDecoder(bytes.NewReader(data), rdf.Turtle)

	// Create a map to store filtered triples
	triples := make(map[rdf.Triple]struct{})

	// Read and filter triples
	for {
		triple, err := decoder.Decode()
		if err != nil {
			if err == rdf.ErrEncoderClosed {
				break
			}
			return nil, err
		}

		// Check if the predicate is one of the desired ones
		if triple.Pred.String() == "http://www.w3.org/2000/01/rdf-schema#label" ||
			triple.Pred.String() == "http://www.w3.org/2004/02/skos/core#prefLabel" ||
			triple.Pred.String() == "http://schema.org/name" ||
			triple.Pred.String() == "http://schema.org/description" ||
			triple.Pred.String() == "http://www.w3.org/2004/02/skos/core#altLabel" {

			// Filter by language tag
			if literal, ok := triple.Obj.(rdf.Literal); ok && literal.Lang() == language {
				triples[triple] = struct{}{}
			}
		}
	}

	// Write filtered triples to buffer
	for triple := range triples {
		if _, err := buffer.WriteString(fmt.Sprintf("%s %s %s .\n", triple.Subj, triple.Pred, triple.Obj)); err != nil {
			return nil, err
		}
	}

	return buffer.Bytes(), nil
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

		if len(line) > 0 && line[len(line)-1] == '.' {
			matches := rdfRegex.FindStringSubmatch(buffer)
			if matches == nil || (len(matches) > 1 && matches[1] == language) {
				// filteredRDF, err := filterRDF([]byte(buffer), language)
				// if err != nil {
				// 	log.Fatalf("Error filtering RDF: %v", err)
				// }

				_, err := gzipWriter.Write([]byte(buffer))
				// _, err = gzipWriter.Write(filteredRDF)
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

func init() {
	cobra.OnInitialize()

	rootCmd.Flags().StringVarP(&language, "language", "l", "en", "Select which language should be filtered")
	rootCmd.Flags().StringVarP(&outDir, "output", "o", "output", "Select output directory")
	rootCmd.Flags().IntVarP(&chunkSize, "chunksize", "s", 3000000, "Select chunk size for splits")
}
