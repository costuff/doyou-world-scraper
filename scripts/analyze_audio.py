#!/usr/bin/env python3
"""
Music Audio Analysis Script
Extracts BPM, key, audio features, and generates spectrograms from MP3 files.
"""

import os
import sys
import json
import librosa
import numpy as np
import soundfile as sf
from pathlib import Path
import argparse
import matplotlib.pyplot as plt
from typing import Dict, Any, Optional


def extract_audio_features(file_path: str) -> Dict[str, Any]:
    """
    Extract comprehensive audio features from an audio file.
    
    Args:
        file_path: Path to the audio file
        
    Returns:
        Dictionary containing all extracted features
    """
    print(f"Analyzing: {file_path}")
    
    # Load audio file
    try:
        y, sr = librosa.load(file_path, sr=None, duration=120)  # Analyze first 2 minutes
    except Exception as e:
        return {"error": f"Failed to load audio: {str(e)}"}
    
    features = {
        "file_path": file_path,
        "file_name": os.path.basename(file_path),
        "duration_seconds": float(librosa.get_duration(y=y, sr=sr))
    }
    
    try:
        # BPM (Tempo)
        tempo, _ = librosa.beat.beat_track(y=y, sr=sr)
        features["bpm"] = float(tempo)
        
        # Key detection (Chroma-based)
        chroma = librosa.feature.chroma_cqt(y=y, sr=sr)
        key_index = np.argmax(np.sum(chroma, axis=1))
        keys = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B']
        features["key"] = keys[key_index]
        
        # Spectral features
        spectral_centroids = librosa.feature.spectral_centroid(y=y, sr=sr)[0]
        features["spectral_centroid_mean"] = float(np.mean(spectral_centroids))
        features["spectral_centroid_std"] = float(np.std(spectral_centroids))
        
        spectral_rolloff = librosa.feature.spectral_rolloff(y=y, sr=sr)[0]
        features["spectral_rolloff_mean"] = float(np.mean(spectral_rolloff))
        
        # Zero crossing rate (useful for percussion/noise detection)
        zcr = librosa.feature.zero_crossing_rate(y)[0]
        features["zero_crossing_rate_mean"] = float(np.mean(zcr))
        
        # RMS Energy
        rms = librosa.feature.rms(y=y)[0]
        features["rms_energy_mean"] = float(np.mean(rms))
        features["rms_energy_std"] = float(np.std(rms))
        
        # MFCC (Mel-frequency cepstral coefficients) - captures timbre
        mfccs = librosa.feature.mfcc(y=y, sr=sr, n_mfcc=13)
        for i in range(13):
            features[f"mfcc_{i}_mean"] = float(np.mean(mfccs[i]))
            features[f"mfcc_{i}_std"] = float(np.std(mfccs[i]))
        
        # Derived features (Shazam-style)
        # Energy: Based on RMS and spectral features
        features["energy"] = float(np.clip(features["rms_energy_mean"] * 2, 0, 1))
        
        # Danceability: Based on tempo stability and rhythm
        tempo_stability = 1.0 - min(float(np.std(_[0])) / 100 if _ else 0 
                                     for _ in [librosa.beat.beat_track(y=y, sr=sr)])
        features["danceability"] = float(np.clip(tempo_stability, 0, 1))
        
        # Acousticness: Lower spectral centroid suggests more acoustic
        features["acousticness"] = float(np.clip(1.0 - (features["spectral_centroid_mean"] / 5000), 0, 1))
        
        # Instrumentalness: Based on spectral characteristics
        features["instrumentalness"] = float(np.clip(1.0 - features["zero_crossing_rate_mean"] * 2, 0, 1))
        
        # Valence: Simplified - based on major/minor and energy
        major_minor_ratio = float(np.sum(chroma[[0, 4, 7], :]) / np.sum(chroma))
        features["valence"] = float(np.clip(major_minor_ratio * 1.2, 0, 1))
        
        # Speechiness: Higher zero crossing rate suggests speech
        features["speechiness"] = float(np.clip(features["zero_crossing_rate_mean"] * 3, 0, 1))
        
        # Liveness: Simplified - based on dynamics variation
        features["liveness"] = float(np.clip(features["rms_energy_std"] * 5, 0, 1))
        
        # Melodicness: Based on harmonic content
        harmonic, _ = librosa.effects.hpss(y)
        harmonic_ratio = float(np.sum(np.abs(harmonic)) / (np.sum(np.abs(y)) + 1e-8))
        features["melodicness"] = float(np.clip(harmonic_ratio, 0, 1))
        
    except Exception as e:
        features["analysis_error"] = str(e)
    
    return features


def generate_spectrogram(file_path: str, output_path: Optional[str] = None) -> Optional[str]:
    """
    Generate a mel spectrogram visualization for an audio file.
    
    Args:
        file_path: Path to the audio file
        output_path: Path to save the spectrogram image (optional)
        
    Returns:
        Path to the saved spectrogram image or None
    """
    try:
        y, sr = librosa.load(file_path, sr=None, duration=120)
        
        # Generate mel spectrogram
        S = librosa.feature.melspectrogram(y=y, sr=sr, n_mels=128, fmax=8000)
        S_dB = librosa.power_to_db(S, ref=np.max)
        
        # Create figure
        plt.figure(figsize=(12, 6))
        librosa.display.specshow(S_dB, sr=sr, x_axis='time', y_axis='mel', fmax=8000)
        plt.colorbar(format='%+2.0f dB')
        plt.title(f'Mel Spectrogram - {os.path.basename(file_path)}')
        plt.tight_layout()
        
        # Save or show
        if output_path:
            plt.savefig(output_path, dpi=150, bbox_inches='tight')
            print(f"Spectrogram saved: {output_path}")
            plt.close()
            return output_path
        else:
            plt.show()
            return None
            
    except Exception as e:
        print(f"Error generating spectrogram: {e}")
        return None


def analyze_directory(directory: str, output_json: str, generate_specs: bool = False):
    """
    Analyze all audio files in a directory.
    
    Args:
        directory: Directory containing audio files
        output_json: Path to save analysis results
        generate_specs: Whether to generate spectrogram images
    """
    results = []
    audio_extensions = {'.mp3', '.wav', '.flac', '.m4a', '.ogg', '.opus'}
    
    # Find all audio files
    audio_files = []
    for root, _, files in os.walk(directory):
        for file in files:
            if Path(file).suffix.lower() in audio_extensions:
                audio_files.append(os.path.join(root, file))
    
    print(f"Found {len(audio_files)} audio files to analyze")
    
    # Analyze each file
    for idx, file_path in enumerate(audio_files, 1):
        print(f"\n[{idx}/{len(audio_files)}] Processing...")
        
        features = extract_audio_features(file_path)
        results.append(features)
        
        # Generate spectrogram if requested
        if generate_specs and "error" not in features:
            spec_dir = os.path.join(directory, "spectrograms")
            os.makedirs(spec_dir, exist_ok=True)
            
            spec_filename = Path(file_path).stem + "_spectrogram.png"
            spec_path = os.path.join(spec_dir, spec_filename)
            
            spectrogram_path = generate_spectrogram(file_path, spec_path)
            if spectrogram_path:
                features["spectrogram_path"] = spectrogram_path
    
    # Save results
    with open(output_json, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"\n✅ Analysis complete! Results saved to: {output_json}")
    print(f"   Analyzed {len(results)} files")
    print(f"   Errors: {sum(1 for r in results if 'error' in r)}")


def main():
    parser = argparse.ArgumentParser(description='Analyze audio files and extract metadata')
    parser.add_argument('input', help='Audio file or directory to analyze')
    parser.add_argument('-o', '--output', help='Output JSON file for results', 
                       default='audio_analysis.json')
    parser.add_argument('-s', '--spectrogram', action='store_true',
                       help='Generate spectrogram images')
    parser.add_argument('--single', action='store_true',
                       help='Analyze single file and print to stdout')
    
    args = parser.parse_args()
    
    if not os.path.exists(args.input):
        print(f"Error: {args.input} does not exist")
        sys.exit(1)
    
    if args.single or os.path.isfile(args.input):
        # Analyze single file
        features = extract_audio_features(args.input)
        print(json.dumps(features, indent=2))
        
        if args.spectrogram:
            generate_spectrogram(args.input)
    else:
        # Analyze directory
        analyze_directory(args.input, args.output, args.spectrogram)


if __name__ == '__main__':
    main()
