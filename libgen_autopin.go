package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/manifoldco/promptui"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
)

// Entry represents a row from the CSV
type Entry struct {
	Dir  string
	Size int
	CID  string
}

// Version number
const version = "0.1.0"

func main() {
	// init
	fmt.Println("Welcome to libgen-autopin!")

	// take args
	app := &cli.App{
		Name:  "libgen-autopin",
		Usage: "easily re-pin libgen on IPFS",
		Flags: []cli.Flag{
			// TODO: Add --source flag
			&cli.IntFlag{
				Name:    "quota",
				Aliases: []string{"q"},
				Value:   50,
				Usage:   "Storage quota allocated for pinning (GB)",
			},
			&cli.StringFlag{
				Name:    "node",
				Aliases: []string{"n"},
				Value:   "http://127.0.0.1:5001",
				Usage:   "IPFS Node",
			},
			&cli.BoolFlag{
				Name:               "version",
				Aliases:            []string{"v"},
				Usage:              "Get version number",
				DisableDefaultText: true,
			},
			&cli.StringFlag{
				Name:    "source",
				Aliases: []string{"s"},
				Value:   "https://pastebin.com/raw/HDVta9Tm",
				Usage:   "FreeRead CID source",
			},
		},
		UsageText: "libgen-autopin [optional flags]",
		Action: func(cCtx *cli.Context) error {
			if cCtx.Bool("version") {
				fmt.Printf("version: %v\n", version)
				os.Exit(0)
			}
			err := repinWithOptions(cCtx.String("node"), cCtx.Int("quota"), cCtx.String("source"))
			if err != nil {
				fmt.Println(err)
				// fmt.Println("libgen-autopin: Invalid usage")
				// fmt.Println("Try libgen-autopin --help for more deatils")
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	// Confirm with user that's what they want to do

	// upload and progress

}

func repinWithOptions(nodeURL string, quota int, source string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	ma, err := ConvertHTTPToMultiaddr(nodeURL)
	if err != nil {
		return fmt.Errorf("failed to convert URI: %w", err)
	}

	fmt.Println("✅ Converted Multiaddress:", ma)
	node, err := rpc.NewApi(ma)
	if err != nil {
		return err
	}

	// Pin a given file by its CID
	ctx := context.Background()

	// Fetch and parse the data
	entries := fetchAndParse(source)

	// Randomly select entries until we fill the quota
	selected, totalSize := randomSelect(entries, quota*1000)

	// Print selected entries and total size
	fmt.Println("✅ Selected Entries:")
	var cids []cid.Cid
	for _, entry := range selected {
		fmt.Printf(" ⮡ Dir: %s, Size: %d MB, CID: %s\n", entry.Dir, entry.Size, entry.CID)
		c, err := cid.Decode(entry.CID)
		if err != nil {
			return err
		}
		cids = append(cids, c)
	}
	fmt.Printf(" ⮕ Total Size: %d GB\n", totalSize)
	// Check with the user if that's ok
	prompt := promptui.Select{
		Label: "Continue [Yes/No]",
		Items: []string{"Yes", "No"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	} else if result == "No" {
		fmt.Println("Process Aborted")
		os.Exit(0)
	}

	fmt.Println("🧘 Have patience, this could take a while!")

	// Then acutally pin
	s.Start()
	s.Suffix = "One sec..."
	for i, cid := range cids {
		s.Suffix = fmt.Sprintf("(%d/%d) pinning: %s", i+1, len(cids), cid.String())
		p := path.FromCid(cid)
		err = node.Pin().Add(ctx, p)
		if err != nil {
			return err
		} else {
			fmt.Println("\nSucsessfully pinned: " + cid.String())
		}
	}
	s.Stop()

	fmt.Println("🥳 Sucsess! Thanks for supporting the network!")

	return nil
}

// fetchAndParse fetches the data from the URL and parses it
func fetchAndParse(url string) []Entry {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	r := csv.NewReader(strings.NewReader(string(body)))
	r.TrimLeadingSpace = true
	r.Comma = ','

	records, err := r.ReadAll()
	if err != nil {
		panic(err)
	}

	var entries []Entry
	for _, record := range records[1:] { // Skip the header
		size, _ := strconv.Atoi(record[1])
		entries = append(entries, Entry{
			Dir:  record[0],
			Size: size,
			CID:  record[2],
		})
	}

	return entries
}

// randomSelect selects random entries until the quota is filled
func randomSelect(entries []Entry, quota int) ([]Entry, int) {
	var selected []Entry
	totalSize := 0
	// this is a counter for failed attempts
	// it allows for continuing the search for CID's to fill the quota better
	c := 0

	for totalSize < quota {
		idx := rand.Intn(len(entries))
		entry := entries[idx]
		if totalSize+entry.Size <= quota {
			selected = append(selected, entry)
			totalSize += entry.Size
		} else {
			c++
			if c > 100 {
				break
			}
		}
	}

	return selected, totalSize / 1000
}

// ConvertHTTPToMultiaddr converts a standard HTTP URI to a multiaddress
func ConvertHTTPToMultiaddr(httpUri string) (multiaddr.Multiaddr, error) {
	// Parse the URI
	parsedUrl, err := url.Parse(httpUri)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}

	// Extract the host and port
	host := parsedUrl.Hostname()
	port := parsedUrl.Port()

	// Default port if none is provided (assuming HTTP)
	if port == "" {
		if parsedUrl.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	// Construct the multiaddress (assuming it's an IPv4 address and using TCP)
	maString := fmt.Sprintf("/ip4/%s/tcp/%s", host, port)
	ma, err := multiaddr.NewMultiaddr(maString)
	if err != nil {
		return nil, fmt.Errorf("error creating multiaddress: %w", err)
	}

	return ma, nil
}
