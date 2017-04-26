package youtube_dl

type AudioDecoder struct{}

func (dec *AudioDecoder) DecodeAudioURLToFile(u url.URL, filename string) error {
	// TODO: Execute youtube-dl and extract audio.
}
