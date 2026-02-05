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
import signal
from functools import wraps

# Timeout decorator
def timeout(seconds=30):
    def decorator(func):
        def _handle_timeout(signum, frame):
            raise TimeoutError(f"Function call timed out after {seconds} seconds")
        
        @wraps(func)
        def wrapper(*args, **kwargs):
            signal.signal(signal.SIGALRM, _handle_timeout)
            signal.alarm(seconds)
            try:
                result = func(*args, **kwargs)
            finally:
                signal.alarm(0)
            return result
        return wrapper
    return decorator


@timeout(60)  # 60 second timeout for loading
def extract_audio_features(file_path: str) -> Dict[str, Any]:
    """
    Extract comprehensive audio features from an audio file.
    
    Args:
        file_path: Path to the audio file
        
    Returns:
        Dictionary containing all extracted features
    """
    print(f"Analyzing: {file_path}", flush=True)
    
    # Load audio file with timeout protection
    try:
        print("Loading audio file...", flush=True)
        y, sr = librosa.load(file_path, sr=22050, duration=120, mono=True)  # Force 22.05kHz, mono
        print(f"Audio loaded: {len(y)/sr:.1f}s at {sr}Hz", flush=True)
    except TimeoutError as e:
        return {"error": f"Audio loading timed out: {str(e)}"}
    except Exception as e:
        return {"error": f"Failed to load audio: {str(e)}"}
    
    features = {
        "file_path": file_path,
        "file_name": os.path.basename(file_path),
        "duration_seconds": float(librosa.get_duration(y=y, sr=sr))
    }
    
    # BPM (Tempo) - Use simpler onset-based method to avoid segfaults
    try:
        print("Extracting BPM...", flush=True)
        # Use onset strength instead of full beat tracking (more stable)
        onset_env = librosa.onset.onset_strength(y=y, sr=sr)
        tempo = librosa.beat.tempo(onset_envelope=onset_env, sr=sr)[0]
        features["bpm"] = float(tempo)
        print(f"BPM: {tempo:.1f}", flush=True)
    except Exception as e:
        print(f"BPM extraction failed: {e}", flush=True)
        features["bpm"] = None
        features["bpm_error"] = str(e)
    
    # Key detection (Chroma-based)
    try:
        print("Detecting key...", flush=True)
        chroma = librosa.feature.chroma_cqt(y=y, sr=sr)
        key_index = np.argmax(np.sum(chroma, axis=1))
        keys = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B']
        features["key"] = keys[key_index]
        print(f"Key: {features['key']}", flush=True)
    except Exception as e:
        print(f"Key detection failed: {e}", flush=True)
        features["key"] = None
        features["key_error"] = str(e)
    
    # Spectral features
    try:
        print("Extracting spectral features...", flush=True)
        spectral_centroids = librosa.feature.spectral_centroid(y=y, sr=sr)[0]
        features["spectral_centroid_mean"] = float(np.mean(spectral_centroids))
        features["spectral_centroid_std"] = float(np.std(spectral_centroids))
        
        spectral_rolloff = librosa.feature.spectral_rolloff(y=y, sr=sr)[0]
        features["spectral_rolloff_mean"] = float(np.mean(spectral_rolloff))
        print("Spectral features extracted", flush=True)
    except Exception as e:
        print(f"Spectral features failed: {e}", flush=True)
        features["spectral_error"] = str(e)
    
    # Zero crossing rate
    try:
        print("Calculating zero crossing rate...", flush=True)
        zcr = librosa.feature.zero_crossing_rate(y)[0]
        features["zero_crossing_rate_mean"] = float(np.mean(zcr))
        print("ZCR calculated", flush=True)
    except Exception as e:
        print(f"ZCR failed: {e}", flush=True)
        features["zcr_error"] = str(e)
    
    # RMS Energy
    try:
        print("Calculating RMS energy...", flush=True)
        rms = librosa.feature.rms(y=y)[0]
        features["rms_energy_mean"] = float(np.mean(rms))
        features["rms_energy_std"] = float(np.std(rms))
        print("RMS energy calculated", flush=True)
    except Exception as e:
        print(f"RMS energy failed: {e}", flush=True)
        features["rms_error"] = str(e)
    
    # MFCC (Mel-frequency cepstral coefficients)
    try:
        print("Extracting MFCCs...", flush=True)
        mfccs = librosa.feature.mfcc(y=y, sr=sr, n_mfcc=13)
        for i in range(13):
            features[f"mfcc_{i}_mean"] = float(np.mean(mfccs[i]))
            features[f"mfcc_{i}_std"] = float(np.std(mfccs[i]))
        print("MFCCs extracted", flush=True)
    except Exception as e:
        print(f"MFCC extraction failed: {e}", flush=True)
        features["mfcc_error"] = str(e)
    
    print("Analysis complete", flush=True)
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
