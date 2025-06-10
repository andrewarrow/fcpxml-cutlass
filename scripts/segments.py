#!/usr/bin/env python3
"""
Intelligent transcript segmentation for video editing.
Breaks transcripts into 18-36 second segments with smart breaking points.
"""

import argparse
import re
from typing import List, Tuple, Optional
from dataclasses import dataclass


@dataclass
class TranscriptLine:
    """Represents a single line from the transcript file."""
    line_number: int
    timestamp: str
    seconds: int
    text: str


@dataclass
class Segment:
    """Represents a segment of the transcript."""
    start_line: int
    end_line: int
    start_time: str
    end_time: str
    start_seconds: int
    end_seconds: int
    duration: int
    text: str


class TranscriptSegmenter:
    """Intelligently segments transcript text based on duration and natural breaks."""
    
    # Transition words and phrases that indicate good break points
    TRANSITION_WORDS = {
        # Strong transitions (weight 3)
        'strong': {
            'now', 'however', 'but', 'meanwhile', 'furthermore', 'moreover',
            'additionally', 'therefore', 'consequently', 'as a result',
            'in conclusion', 'to summarize', 'moving on', 'next up',
            'speaking of', 'that said', 'on the other hand', 'in contrast',
            'alternatively', 'instead', 'rather', 'anyway', 'so'
        },
        # Medium transitions (weight 2)
        'medium': {
            'also', 'and', 'plus', 'then', 'after', 'before', 'while',
            'during', 'since', 'because', 'if', 'when', 'where', 'why',
            'how', 'what', 'which', 'who', 'though', 'although', 'unless',
            'until', 'once', 'as', 'like'
        },
        # Weak transitions (weight 1)
        'weak': {
            'the', 'this', 'that', 'these', 'those', 'here', 'there',
            'i', 'you', 'we', 'they', 'it', 'he', 'she'
        }
    }
    
    # Phrases that suggest continuation (avoid breaking here)
    CONTINUATION_PHRASES = {
        'for example', 'such as', 'in other words', 'that is',
        'i.e.', 'e.g.', 'specifically', 'particularly', 'especially',
        'including', 'like this', 'as follows', 'as well as'
    }
    
    def __init__(self, min_duration: int = 18, max_duration: int = 36):
        self.min_duration = min_duration
        self.max_duration = max_duration
    
    def parse_transcript_file(self, filepath: str) -> List[TranscriptLine]:
        """Parse the .codes transcript file into structured data."""
        lines = []
        line_counter = 1
        
        with open(filepath, 'r', encoding='utf-8') as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                
                # Parse format: timestamp seconds text
                parts = line.split('\t')
                if len(parts) >= 3:
                    timestamp = parts[0].strip()
                    seconds = int(parts[1].strip())
                    text = '\t'.join(parts[2:]).strip()  # Rejoin text in case it has tabs
                    
                    lines.append(TranscriptLine(line_counter, timestamp, seconds, text))
                    line_counter += 1
        
        return lines
    
    def calculate_break_score(self, line: TranscriptLine, next_line: Optional[TranscriptLine]) -> float:
        """Calculate how good a break point this line would be (higher = better)."""
        score = 0.0
        text = line.text.lower()
        
        # Check for strong transition words
        for word in self.TRANSITION_WORDS['strong']:
            if word in text:
                score += 3.0
        
        # Check for medium transition words
        for word in self.TRANSITION_WORDS['medium']:
            if word in text:
                score += 2.0
        
        # Check for weak transition words
        for word in self.TRANSITION_WORDS['weak']:
            if text.startswith(word + ' '):
                score += 1.0
        
        # Penalize continuation phrases
        for phrase in self.CONTINUATION_PHRASES:
            if phrase in text:
                score -= 2.0
        
        # Bonus for natural sentence endings (even without punctuation)
        if re.search(r'\b(right|okay|well|yeah|alright|so)\s*$', text):
            score += 1.5
        
        # Bonus for topic changes (mentioning new features, apps, etc.)
        topic_words = ['app', 'feature', 'update', 'new', 'also', 'another', 'next']
        if next_line and any(word in next_line.text.lower() for word in topic_words):
            score += 1.0
        
        # Bonus for ending with certain words that suggest completion
        ending_words = ['now', 'too', 'well', 'right', 'there', 'here', 'done']
        if any(text.strip().endswith(' ' + word) for word in ending_words):
            score += 0.5
        
        return score
    
    def find_best_break_in_range(self, lines: List[TranscriptLine], start_idx: int, 
                                 min_end_idx: int, max_end_idx: int) -> int:
        """Find the best break point within the given range."""
        best_idx = min_end_idx
        best_score = -1.0
        
        for i in range(min_end_idx, min(max_end_idx + 1, len(lines))):
            next_line = lines[i + 1] if i + 1 < len(lines) else None
            score = self.calculate_break_score(lines[i], next_line)
            
            if score > best_score:
                best_score = score
                best_idx = i
        
        return best_idx
    
    def create_segments(self, lines: List[TranscriptLine]) -> List[Segment]:
        """Create segments from transcript lines with intelligent breaking."""
        segments = []
        current_start = 0
        
        while current_start < len(lines):
            start_line = lines[current_start]
            
            # Find the minimum and maximum end points based on duration
            min_end_idx = current_start
            max_end_idx = current_start
            
            # Find minimum duration boundary
            for i in range(current_start, len(lines)):
                duration = lines[i].seconds - start_line.seconds
                if duration >= self.min_duration:
                    min_end_idx = i
                    break
            else:
                # If we can't reach min_duration, use what we have
                min_end_idx = len(lines) - 1
            
            # Find maximum duration boundary
            for i in range(min_end_idx, len(lines)):
                duration = lines[i].seconds - start_line.seconds
                if duration > self.max_duration:
                    max_end_idx = i - 1
                    break
                max_end_idx = i
            
            # If we have room to choose, find the best break point
            if max_end_idx > min_end_idx:
                end_idx = self.find_best_break_in_range(lines, current_start, min_end_idx, max_end_idx)
            else:
                end_idx = min_end_idx
            
            # Ensure we don't go past the end
            end_idx = min(end_idx, len(lines) - 1)
            
            # Create the segment
            end_line = lines[end_idx]
            duration = end_line.seconds - start_line.seconds
            
            # Collect all text in this segment
            segment_text = ' '.join(line.text for line in lines[current_start:end_idx + 1])
            
            segment = Segment(
                start_line=start_line.line_number,
                end_line=end_line.line_number,
                start_time=start_line.timestamp,
                end_time=end_line.timestamp,
                start_seconds=start_line.seconds,
                end_seconds=end_line.seconds,
                duration=duration,
                text=segment_text
            )
            
            segments.append(segment)
            current_start = end_idx + 1
        
        return segments
    
    def format_time(self, seconds: int) -> str:
        """Convert seconds to MM:SS format."""
        minutes = seconds // 60
        secs = seconds % 60
        return f"{minutes}:{secs:02d}"
    
    def display_segments(self, segments: List[Segment]):
        """Display the segments with timecodes and duration."""
        print(f"\n{'='*80}")
        print(f"TRANSCRIPT SEGMENTS ({len(segments)} total)")
        print(f"{'='*80}")
        
        total_duration = 0
        
        for i, segment in enumerate(segments, 1):
            total_duration += segment.duration
            
            print(f"\nSegment {i}:")
            print(f"  Time: {segment.start_time} - {segment.end_time} ({segment.duration}s)")
            print(f"  Lines: {segment.start_line} - {segment.end_line}")
            print(f"  Text: {segment.text[:100]}{'...' if len(segment.text) > 100 else ''}")
        
        print(f"\n{'='*80}")
        print(f"SUMMARY")
        print(f"{'='*80}")
        print(f"Total segments: {len(segments)}")
        print(f"Total duration: {self.format_time(total_duration)} ({total_duration}s)")
        if len(segments) > 0:
            print(f"Average segment length: {total_duration / len(segments):.1f}s")
        else:
            print("Average segment length: N/A")
        
        # Duration distribution
        short = sum(1 for s in segments if s.duration < 18)
        optimal = sum(1 for s in segments if 18 <= s.duration <= 36)
        long = sum(1 for s in segments if s.duration > 36)
        
        print(f"Duration distribution:")
        print(f"  Short (<18s): {short}")
        print(f"  Optimal (18-36s): {optimal}")
        print(f"  Long (>36s): {long}")


def main():
    parser = argparse.ArgumentParser(description='Intelligently segment transcript files')
    parser.add_argument('file', help='Path to the .codes transcript file')
    parser.add_argument('--min-duration', type=int, default=18, 
                       help='Minimum segment duration in seconds (default: 18)')
    parser.add_argument('--max-duration', type=int, default=36,
                       help='Maximum segment duration in seconds (default: 36)')
    
    args = parser.parse_args()
    
    segmenter = TranscriptSegmenter(args.min_duration, args.max_duration)
    
    try:
        lines = segmenter.parse_transcript_file(args.file)
        print(f"Loaded {len(lines)} transcript lines from {args.file}")
        
        segments = segmenter.create_segments(lines)
        segmenter.display_segments(segments)
        
    except FileNotFoundError:
        print(f"Error: File '{args.file}' not found")
    except Exception as e:
        print(f"Error processing file: {e}")


if __name__ == '__main__':
    main()