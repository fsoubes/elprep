package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/exascience/elprep/internal"
	"github.com/exascience/elprep/sam"
)

const SplitHelp = "Split parameters:\n" +
	"elprep split (sam-file | /path/to/input/) /path/to/output/\n" +
	"[--output-prefix name]\n" +
	"[--output-type [sam | bam | cram]]\n" +
	"[--nr-of-threads nr]\n" +
	"[--reference-t fai-file]\n" +
	"[--reference-T fasta-file]\n"

/*
Split implements the elprep split command.
*/
func Split() error {
	var (
		outputPrefix, outputType, reference_t, reference_T string
		nrOfThreads                                        int
	)

	var flags flag.FlagSet

	flags.StringVar(&outputPrefix, "output-prefix", "", "prefix for the output files")
	flags.StringVar(&outputType, "output-type", "", "format of the output files")
	flags.IntVar(&nrOfThreads, "nr-of-threads", 0, "number of worker threads")
	flags.StringVar(&reference_t, "reference-t", "", "specify a .fai file for cram output")
	flags.StringVar(&reference_T, "reference-T", "", "specify a .fasta file for cram output")

	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Incorrect number of parameters.")
		fmt.Fprint(os.Stderr, SplitHelp)
		os.Exit(1)
	}

	input := getFilename(os.Args[2], SplitHelp)
	output := getFilename(os.Args[3], SplitHelp)

	if err := flags.Parse(os.Args[4:]); err != nil {
		x := 0
		if err != flag.ErrHelp {
			fmt.Fprintln(os.Stderr, err.Error())
			x = 1
		}
		fmt.Fprint(os.Stderr, SplitHelp)
		os.Exit(x)
	}

	ext := filepath.Ext(input)
	if outputPrefix == "" {
		base := filepath.Base(input)
		outputPrefix = base[:len(base)-len(ext)]
	}
	if outputType == "" {
		switch ext {
		case ".sam", ".bam", ".cram":
			outputType = ext[1:]
		default:
			outputType = "sam"
		}
	}

	setLogOutput()

	// sanity checks

	sanityChecksFailed := false

	reference_t, reference_T, success := checkCramOutputOptions(outputType, reference_t, reference_T)
	sanityChecksFailed = !success

	if filepath.Dir(output) != filepath.Clean(output) {
		log.Printf("Given output path is not a path: %v.\n", output)
		sanityChecksFailed = true
	}

	if nrOfThreads < 0 {
		sanityChecksFailed = true
		log.Println("Error: Invalid nr-of-threads: ", nrOfThreads)
	}

	if sanityChecksFailed {
		fmt.Fprint(os.Stderr, SplitHelp)
		os.Exit(1)
	}

	// building output command line

	var command bytes.Buffer
	fmt.Fprint(&command, os.Args[0], " split ", input, " ", output)
	fmt.Fprint(&command, " --output-prefix ", outputPrefix)
	fmt.Fprint(&command, " --output-type ", outputType)
	if nrOfThreads > 0 {
		runtime.GOMAXPROCS(nrOfThreads)
		fmt.Fprint(&command, " --nr-of-threads ", nrOfThreads)
	}
	if reference_t != "" {
		fmt.Fprint(&command, " --reference-t ", reference_t)
	}
	if reference_T != "" {
		fmt.Fprint(&command, " --reference-T ", reference_T)
	}

	// executing command

	log.Println("Executing command:\n", command.String())

	fullInput, err := internal.FullPathname(input)
	if err != nil {
		return err
	}

	fullOutput, err := internal.FullPathname(output)
	if err != nil {
		return err
	}

	err = os.MkdirAll(output, 0700)
	if err != nil {
		return err
	}

	return sam.SplitFilePerChromosome(fullInput, fullOutput, outputPrefix, outputType, reference_t, reference_T)
}
