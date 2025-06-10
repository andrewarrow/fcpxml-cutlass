#!/usr/bin/env python3

import sys
import os
from bs4 import BeautifulSoup
import re

def parse_timestamp_to_seconds(timestamp):
    """Convert timestamp (e.g., '1:23') to total seconds."""
    parts = timestamp.split(':')
    if len(parts) == 2:
        minutes, seconds = parts
        return int(minutes) * 60 + int(seconds)
    elif len(parts) == 3:
        hours, minutes, seconds = parts
        return int(hours) * 3600 + int(minutes) * 60 + int(seconds)
    return 0

def extract_transcript(video_id):
    """Extract timecode and text from YouTube transcript HTML file."""
    html_file = f"../data/{video_id}.html"
    
    if not os.path.exists(html_file):
        print(f"Error: File {html_file} not found")
        return
    
    with open(html_file, 'r', encoding='utf-8') as f:
        html_content = f.read()
    
    soup = BeautifulSoup(html_content, 'html.parser')
    
    # Find all transcript segments
    segments = soup.find_all('ytd-transcript-segment-renderer')
    
    for segment in segments:
        # Extract timestamp
        timestamp_elem = segment.find('div', class_='segment-timestamp')
        if timestamp_elem:
            timestamp = timestamp_elem.get_text().strip()
        else:
            timestamp = ""
        
        # Extract text
        text_elem = segment.find('yt-formatted-string', class_='segment-text')
        if text_elem:
            text = text_elem.get_text().strip()
        else:
            text = ""
        
        if timestamp and text:
            total_seconds = parse_timestamp_to_seconds(timestamp)
            print(f"{timestamp}\t{total_seconds}\t{text}")

def main():
    if len(sys.argv) != 2:
        print("Usage: python transcript.py <video_id>")
        print("Example: python transcript.py MOCBOaYMYvk")
        sys.exit(1)
    
    video_id = sys.argv[1]
    extract_transcript(video_id)

if __name__ == "__main__":
    main()
