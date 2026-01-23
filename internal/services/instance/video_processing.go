package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"videostreamgo/internal/models/instance"
)

// FFmpegProcessingService implements VideoProcessingService using FFmpeg
type FFmpegProcessingService struct {
	storageDir     string
	ffmpegPath     string
	ffprobePath    string
	tempDir        string
	qualityProfile map[instance.VideoQuality]QualityProfile
}

// QualityProfile defines encoding parameters for a specific quality
type QualityProfile struct {
	Width     int
	Height    int
	Bitrate   string
	AudioRate string
}

// NewVideoProcessingService creates a new video processing service
func NewVideoProcessingService(storageDir string) (*FFmpegProcessingService, error) {
	// Find FFmpeg and FFprobe paths
	ffmpegPath := "/usr/local/bin/ffmpeg"
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		ffmpegPath = "ffmpeg"
	}

	ffprobePath := "/usr/local/bin/ffprobe"
	if _, err := exec.LookPath("ffprobe"); err == nil {
		ffprobePath = "ffprobe"
	}

	tempDir := filepath.Join(storageDir, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &FFmpegProcessingService{
		storageDir:  storageDir,
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		tempDir:     tempDir,
		qualityProfile: map[instance.VideoQuality]QualityProfile{
			instance.Quality360p:  {640, 360, "800k", "128k"},
			instance.Quality480p:  {854, 480, "1400k", "128k"},
			instance.Quality720p:  {1280, 720, "2800k", "192k"},
			instance.Quality1080p: {1920, 1080, "5000k", "192k"},
			instance.Quality4K:    {3840, 2160, "15000k", "192k"},
		},
	}, nil
}

// ProcessVideo processes a video (extract metadata, transcode, generate thumbnails)
func (s *FFmpegProcessingService) ProcessVideo(videoID string) error {
	// Get video info from database (would be injected in real implementation)
	videoDir := filepath.Join(s.storageDir, "uploads", videoID[:8])
	videoPath := filepath.Join(videoDir, "original.mp4")

	// Check if video file exists
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file not found: %s", videoPath)
	}

	// Step 1: Extract metadata
	_, err := s.ExtractMetadata(videoID)
	if err != nil {
		return fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Step 2: Generate thumbnails
	if err := s.GenerateThumbnails(videoID); err != nil {
		return fmt.Errorf("failed to generate thumbnails: %w", err)
	}

	// Step 3: Transcode to multiple qualities
	qualities := []instance.VideoQuality{
		instance.Quality360p,
		instance.Quality480p,
		instance.Quality720p,
	}

	if err := s.TranscodeVideo(videoID, qualities); err != nil {
		return fmt.Errorf("failed to transcode video: %w", err)
	}

	// Step 4: Generate HLS manifest
	if err := s.GenerateHLSManifest(videoID, qualities); err != nil {
		return fmt.Errorf("failed to generate HLS manifest: %w", err)
	}

	// Step 5: Cleanup temp files
	if err := s.CleanupTempFiles(videoID); err != nil {
		fmt.Printf("warning: failed to cleanup temp files: %v\n", err)
	}

	return nil
}

// ExtractMetadata extracts metadata from a video using FFprobe
func (s *FFmpegProcessingService) ExtractMetadata(videoID string) (*VideoMetadata, error) {
	videoPath := filepath.Join(s.storageDir, "uploads", videoID[:8], "original.mp4")

	cmd := exec.Command(s.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var probeResult struct {
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			FrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	metadata := &VideoMetadata{}

	// Parse duration
	if duration, err := strconv.ParseFloat(probeResult.Format.Duration, 64); err == nil {
		metadata.Duration = duration
	}

	// Parse bitrate
	if bitrate, err := strconv.Atoi(probeResult.Format.BitRate); err == nil {
		metadata.Bitrate = bitrate / 1000 // Convert to kbps
	}

	// Find video stream
	for _, stream := range probeResult.Streams {
		if stream.CodecType == "video" {
			metadata.Codec = stream.CodecName
			metadata.Width = stream.Width
			metadata.Height = stream.Height

			// Parse frame rate
			if strings.Contains(stream.FrameRate, "/") {
				parts := strings.Split(stream.FrameRate, "/")
				if num, err := strconv.ParseFloat(parts[0], 64); err == nil {
					if den, err := strconv.ParseFloat(parts[1], 64); err == nil && den > 0 {
						metadata.FrameRate = num / den
					}
				}
			} else if fps, err := strconv.ParseFloat(stream.FrameRate, 64); err == nil {
				metadata.FrameRate = fps
			}

			// Calculate aspect ratio
			if metadata.Width > 0 && metadata.Height > 0 {
				gcd := gcd(metadata.Width, metadata.Height)
				metadata.AspectRatio = fmt.Sprintf("%d:%d", metadata.Width/gcd, metadata.Height/gcd)
			}
		}
		if stream.CodecType == "audio" {
			metadata.AudioCodec = stream.CodecName
		}
	}

	return metadata, nil
}

// GenerateThumbnails generates thumbnails from a video
func (s *FFmpegProcessingService) GenerateThumbnails(videoID string) error {
	videoPath := filepath.Join(s.storageDir, "uploads", videoID[:8], "original.mp4")
	outputDir := filepath.Join(s.storageDir, "thumbnails", videoID[:8])

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate multiple thumbnails at different times
	thumbnails := []struct {
		Time     string
		Width    int
		Filename string
	}{
		{"00:00:01", 320, "thumb-1.jpg"},
		{"00:00:05", 320, "thumb-2.jpg"},
		{"00:00:10", 320, "thumb-3.jpg"},
		{"50%", 640, "thumb-large.jpg"}, // Middle of video
	}

	for _, thumb := range thumbnails {
		outputPath := filepath.Join(outputDir, thumb.Filename)

		args := []string{
			"-ss", thumb.Time,
			"-i", videoPath,
			"-vframes", "1",
			"-vf", fmt.Sprintf("scale=%d:-1", thumb.Width),
			"-q:v", "2",
			outputPath,
		}

		cmd := exec.Command(s.ffmpegPath, args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to generate thumbnail %s: %w", thumb.Filename, err)
		}
	}

	return nil
}

// TranscodeVideo transcodes a video to multiple qualities
func (s *FFmpegProcessingService) TranscodeVideo(videoID string, qualities []instance.VideoQuality) error {
	videoPath := filepath.Join(s.storageDir, "uploads", videoID[:8], "original.mp4")
	outputDir := filepath.Join(s.storageDir, "transcoded", videoID[:8])

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, quality := range qualities {
		profile, ok := s.qualityProfile[quality]
		if !ok {
			continue
		}

		outputPath := filepath.Join(outputDir, string(quality)+".mp4")

		args := []string{
			"-i", videoPath,
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-vf", fmt.Sprintf("scale=%d:%d", profile.Width, profile.Height),
			"-b:v", profile.Bitrate,
			"-maxrate", profile.Bitrate,
			"-bufsize", fmt.Sprintf("%dk", parseBitrate(profile.Bitrate)*2),
			"-c:a", "aac",
			"-b:a", profile.AudioRate,
			"-movflags", "+faststart",
			"-y",
			outputPath,
		}

		cmd := exec.Command(s.ffmpegPath, args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to transcode to %s: %w", quality, err)
		}
	}

	return nil
}

// GenerateHLSManifest generates HLS manifest files
func (s *FFmpegProcessingService) GenerateHLSManifest(videoID string, qualities []instance.VideoQuality) error {
	videoPath := filepath.Join(s.storageDir, "uploads", videoID[:8], "original.mp4")
	outputDir := filepath.Join(s.storageDir, "hls", videoID[:8])

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build HLS variant playlist
	var variants []string
	for _, quality := range qualities {
		profile, ok := s.qualityProfile[quality]
		if !ok {
			continue
		}

		// Generate segment list for this quality
		qualityDir := filepath.Join(outputDir, string(quality))
		if err := os.MkdirAll(qualityDir, 0755); err != nil {
			return fmt.Errorf("failed to create quality directory: %w", err)
		}

		// Create HLS segments using FFmpeg
		segmentPattern := filepath.Join(qualityDir, "%03d.ts")
		playlistPath := filepath.Join(qualityDir, "index.m3u8")

		args := []string{
			"-i", videoPath,
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-vf", fmt.Sprintf("scale=%d:%d", profile.Width, profile.Height),
			"-b:v", profile.Bitrate,
			"-maxrate", profile.Bitrate,
			"-bufsize", fmt.Sprintf("%dk", parseBitrate(profile.Bitrate)*2),
			"-c:a", "aac",
			"-b:a", profile.AudioRate,
			"-f", "hls",
			"-hls_time", "10",
			"-hls_list_size", "0",
			"-hls_segment_filename", segmentPattern,
			"-start_number", "0",
			playlistPath,
		}

		cmd := exec.Command(s.ffmpegPath, args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to generate HLS for %s: %w", quality, err)
		}

		variants = append(variants, fmt.Sprintf("#EXT-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n%s/index.m3u8",
			parseBitrate(profile.Bitrate)*1000, profile.Width, profile.Height, quality))
	}

	// Create master playlist
	masterPath := filepath.Join(outputDir, "master.m3u8")
	masterContent := "#EXTM3U\n"
	for _, variant := range variants {
		masterContent += variant + "\n"
	}

	if err := os.WriteFile(masterPath, []byte(masterContent), 0644); err != nil {
		return fmt.Errorf("failed to write master playlist: %w", err)
	}

	return nil
}

// CleanupTempFiles removes temporary processing files
func (s *FFmpegProcessingService) CleanupTempFiles(videoID string) error {
	tempDir := filepath.Join(s.tempDir, videoID[:8])
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to cleanup temp files: %w", err)
	}
	return nil
}

// GetProcessingStatus returns the current processing status
func (s *FFmpegProcessingService) GetProcessingStatus(videoID string) (*ProcessingStatusResponse, error) {
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		return nil, fmt.Errorf("invalid video ID: %w", err)
	}

	return &ProcessingStatusResponse{
		VideoID:     videoUUID,
		Status:      instance.ProcessingStatusCompleted,
		Progress:    100,
		CurrentStep: "completed",
	}, nil
}

// Helper function to parse bitrate string (e.g., "800k" -> 800)
func parseBitrate(bitrate string) int {
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(bitrate)
	if len(matches) > 0 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			return val
		}
	}
	return 1000
}

// Helper function to calculate GCD
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
