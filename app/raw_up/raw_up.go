package raw_up

import (
	"context"
	"encoding/json"
	"os"

	"github.com/go-faster/errors"
	"github.com/gotd/td/telegram"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/viper"
	"go.uber.org/multierr"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/iyear/tdl/core/dcpool"
	"github.com/iyear/tdl/core/storage"
	"github.com/iyear/tdl/core/tclient"
	"github.com/iyear/tdl/pkg/consts"
)

// CustomProgressTracker implements the Progress interface to track upload progress.
type CustomProgressTracker struct {
	bar *progressbar.ProgressBar
}

// NewCustomProgressTracker creates a new CustomProgressTracker with an initialized progress bar.
func NewCustomProgressTracker(totalSize int64) *CustomProgressTracker {
	// Initialize the progress bar to write to stderr
	bar := progressbar.NewOptions64(totalSize, progressbar.OptionSetWriter(os.Stderr), progressbar.OptionShowBytes(true), progressbar.OptionSetDescription("Uploading"))
	return &CustomProgressTracker{bar: bar}
}

// Chunk updates the progress bar based on the current upload progress.
func (c *CustomProgressTracker) Chunk(ctx context.Context, state uploader.ProgressState) error {
	// Set the progress bar to the current uploaded size.
	_ = c.bar.Set64(state.Uploaded)
	return nil
}

type Options struct {
	Path string
	Name string
}

type UploadResult struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Parts       int    `json:"parts"`
	MD5Checksum string `json:"md5_checksum"`
	IsBigFile   bool   `json:"is_big_file"`
}

func Run(ctx context.Context, c *telegram.Client, kvd storage.Storage, opts Options) (rerr error) {
	// Check if the file exists
	stat, err := os.Stat(opts.Path)
	if err != nil {
		return errors.Wrap(err, "stat file")
	}

	pool := dcpool.NewPool(c,
		int64(viper.GetInt(consts.FlagPoolSize)),
		tclient.NewDefaultMiddlewares(ctx, viper.GetDuration(consts.FlagReconnectTimeout))...)
	defer multierr.AppendInvoke(&rerr, multierr.Close(pool))

	up := uploader.NewUploader(pool.Default(ctx)).
		WithPartSize(viper.GetInt(consts.FlagPartSize)).
		WithThreads(viper.GetInt(consts.FlagThreads)).
		WithProgress(NewCustomProgressTracker(stat.Size()))

	f, err := os.Open(opts.Path)
	if err != nil {
		return errors.Wrap(err, "open file")
	}

	uploaded, err := up.Upload(ctx, uploader.NewUpload(opts.Name, f, stat.Size()))
	if err != nil {
		return errors.Wrap(err, "upload file")
	}

	result := &UploadResult{}

	switch uploaded := uploaded.(type) {
	case *tg.InputFile:
		result.MD5Checksum = uploaded.GetMD5Checksum()
		result.ID = uploaded.GetID()
		result.Name = uploaded.GetName()
		result.Parts = uploaded.GetParts()
		result.IsBigFile = false
	case *tg.InputFileBig:
		result.ID = uploaded.GetID()
		result.Name = uploaded.GetName()
		result.Parts = uploaded.GetParts()
		result.IsBigFile = true
	}

	// Print out the json of result in std out using golang std
	json.NewEncoder(os.Stdout).Encode(result)

	return nil
}
