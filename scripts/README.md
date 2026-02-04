# Music Audio Analysis Scripts

This directory contains Python scripts for analyzing audio files and extracting metadata.

## Setup

1. Create a Python virtual environment (already done if using VS Code):
```bash
python3 -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate
```

2. Install dependencies:
```bash
pip install -r scripts/requirements.txt
```

## Scripts

### `analyze_audio.py`

Extracts comprehensive audio features from MP3/audio files including:
- **BPM (Tempo)**: Beats per minute
- **Musical Key**: Detected key (C, C#, D, etc.)
- **Audio Features** (Shazam-style):
  - Energy - Intensity and dynamics
  - Danceability - Rhythm stability and tempo
  - Acousticness - Acoustic vs electronic
  - Valence - Musical positivity/mood
  - Melodicness - Harmonic content
  - Instrumentalness - Vocal vs instrumental
  - Speechiness - Spoken word presence
  - Liveness - Live performance indicators
- **Spectral Features**: Centroid, rolloff, zero-crossing rate, RMS energy
- **MFCCs**: 13 Mel-frequency cepstral coefficients (timbre characteristics)

#### Usage

**Analyze a single file:**
```bash
python scripts/analyze_audio.py --single "path/to/song.mp3"
```

**Analyze all files in a directory:**
```bash
python scripts/analyze_audio.py data/downloads -o analysis_results.json
```

**Generate spectrograms:**
```bash
python scripts/analyze_audio.py data/downloads -o results.json --spectrogram
```

This will create a `spectrograms/` folder with mel spectrogram images for each track.

#### Output Format

JSON file with array of track analyses:
```json
[
  {
    "file_path": "/path/to/song.mp3",
    "file_name": "Song Name [ID].mp3",
    "duration_seconds": 245.6,
    "bpm": 128.5,
    "key": "Am",
    "energy": 0.85,
    "danceability": 0.92,
    "acousticness": 0.12,
    "valence": 0.68,
    "melodicness": 0.75,
    "instrumentalness": 0.95,
    "speechiness": 0.03,
    "liveness": 0.15,
    "spectral_centroid_mean": 2500.5,
    "rms_energy_mean": 0.12,
    "mfcc_0_mean": -210.5,
    ...
  }
]
```

## Examples

```bash
# Analyze one track and print results
python scripts/analyze_audio.py --single "data/downloads/Beautifulness [5UrseUmU1oI].mp3"

# Analyze all tracks in downloads folder
python scripts/analyze_audio.py data/downloads -o data/audio_analysis.json

# Analyze with spectrogram generation (takes longer)
python scripts/analyze_audio.py data/downloads -o data/audio_analysis.json --spectrogram
```

## Integration with API

These analysis results can be:
1. Stored in PostgreSQL database (future phase)
2. Exposed via REST API endpoints
3. Used for music library search/filtering
4. Visualized in the frontend dashboard

## Technical Notes

- Analysis processes first 2 minutes of each track (configurable)
- Uses librosa for audio feature extraction
- Spectrograms are mel spectrograms (frequency on mel scale)
- Feature values normalized to 0-1 range where applicable
- Compatible with MP3, WAV, FLAC, M4A, OGG, Opus formats
