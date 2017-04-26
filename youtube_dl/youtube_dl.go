package youtube_dl

import (
	"net/url"
)

type AudioDecoder struct{}

func (dec *AudioDecoder) DecodeAudioURLToFile(u url.URL, filename string) error {
	panic("TODO: Execute youtube-dl and extract audio.")
}
