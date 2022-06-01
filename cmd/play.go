package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/spf13/cobra"
)

func getLargestFile(t *torrent.Torrent) *torrent.File {
	var target *torrent.File
	var maxSize int64

	for _, file := range t.Files() {
		if maxSize < file.Length() {
			maxSize = file.Length()
			target = file
		}
	}

	return target
}

func percentage(t *torrent.Torrent) float64 {
	info := t.Info()

	if info == nil {
		return 0
	}

	return float64(t.BytesCompleted()) / float64(info.TotalLength()) * 100
}

// playCmd represents the play command
var playCmd = &cobra.Command{
	Use:   "play [magnet link]",
	Short: "Play a magnet link",
	Run: func(cmd *cobra.Command, args []string) {
		u, err := url.ParseRequestURI(args[0])
		if err != nil {
			panic(err)
		}

		if u.Scheme != "magnet" {
			panic("invalid magnet link")
		}

		// port := cmd.Flag("port").Value.String()
		magnet := u.String()

		dataDir := filepath.Join(os.TempDir(), "peertv-"+u.Query()["xt"][0])

		err = os.Mkdir(dataDir, 0755)
		if err != nil && os.IsNotExist(err) {
			panic(err)
		}

		fmt.Println("Loading torrent...")

		clientConfig := torrent.NewDefaultClientConfig()
		clientConfig.DataDir = dataDir

		client, err := torrent.NewClient(clientConfig)
		if err != nil {
			panic(err)
		}
		defer client.Close()
		t, err := client.AddMagnet(magnet)
		if err != nil {
			panic(err)
		}
		<-t.GotInfo()
		t.DownloadAll()

		largestFilePath := filepath.Join(dataDir, getLargestFile(t).Path())

		// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 	http.ServeFile(w, r, largestFilePath)
		// })
		// go http.ListenAndServe(":"+port, nil)
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			exec.Command("mpv", largestFilePath).Run()
			wg.Done()
		}()

		i := 0
		for percentage(t) != 100 || i == 0 {
			time.Sleep(200 * time.Millisecond)
			fmt.Print("\033[H\033[2J") // clear screen
			fmt.Println("File location: ", largestFilePath)
			fmt.Println("Torrent name", t.Name())
			fmt.Println("Download progress", percentage(t), "%")
			fmt.Println("Peers", len(t.PeerConns()))
			i++
		}

		client.WaitAll()
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(playCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// playCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// playCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// playCmd.Flags().StringP("port", "p", "8888", "Port to use")
}
