package cmd

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/knakk/rdf"
)

func regexFilterRDF(rdf string) string {
	// Define regex patterns for filtering based on @en
	labelPattern := regexp.MustCompile(`(rdfs:label|skos:prefLabel|schema:name)\s+"[^"]+"@(\w+)\s*;`)
	descPattern := regexp.MustCompile(`schema:description\s+"[^"]+"@(\w+)(?:\s*,\s*|\s*;)`)
	altLabelPattern := regexp.MustCompile(`skos:altLabel\s+"[^"]+"@(\w+)(?:\s*,\s*|\s*\.)`)

	// Filter labels with @en
	rdf = labelPattern.ReplaceAllStringFunc(rdf, func(match string) string {
		if strings.Contains(match, `@en`) {
			return match
		}
		return ""
	})

	// Filter descriptions with @en
	rdf = descPattern.ReplaceAllStringFunc(rdf, func(match string) string {
		if strings.Contains(match, `@en`) {
			return match
		}
		return ""
	})

	// Filter altLabel with @en
	rdf = altLabelPattern.ReplaceAllStringFunc(rdf, func(match string) string {
		if strings.Contains(match, `@en`) {
			return match
		}
		return ""
	})

	// Clean up any remaining commas, semicolons, or trailing spaces from filtering
	rdf = strings.TrimSpace(rdf)
	rdf = strings.ReplaceAll(rdf, "\n\n", "\n")
	return rdf
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
