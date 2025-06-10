#!/usr/bin/env python3

import sys
import os
try:
    import termios
    import tty
    TERMIOS_AVAILABLE = True
except ImportError:
    TERMIOS_AVAILABLE = False

class TranscriptSelector:
    def __init__(self, lines):
        self.lines = []
        self.parse_lines(lines)
        self.current_index = 0
        self.selected_ranges = []
        self.selecting = False
        self.selection_start = None
        
    def parse_lines(self, lines):
        """Parse transcript lines into structured data."""
        for line in lines:
            line = line.strip()
            if not line:
                continue
            parts = line.split('\t')
            if len(parts) >= 3:
                timestamp = parts[0]
                seconds = int(parts[1])
                text = '\t'.join(parts[2:])  # Join remaining parts in case text contains tabs
                self.lines.append({
                    'timestamp': timestamp,
                    'seconds': seconds,
                    'text': text
                })
    
    def get_duration_for_range(self, start_idx, end_idx):
        """Calculate duration for a range of lines."""
        if start_idx >= len(self.lines) or end_idx >= len(self.lines):
            return 0
        
        start_seconds = self.lines[start_idx]['seconds']
        
        # Find the next timestamp after the end to calculate duration
        if end_idx + 1 < len(self.lines):
            end_seconds = self.lines[end_idx + 1]['seconds']
        else:
            # For the last segment, estimate 3 seconds duration
            end_seconds = self.lines[end_idx]['seconds'] + 3
            
        return end_seconds - start_seconds
    
    def get_total_selected_duration(self):
        """Calculate total duration of all selected ranges."""
        total = 0
        for start_idx, end_idx in self.selected_ranges:
            total += self.get_duration_for_range(start_idx, end_idx)
        return total
    
    def is_line_selected(self, idx):
        """Check if a line is part of any selected range."""
        for start_idx, end_idx in self.selected_ranges:
            if start_idx <= idx <= end_idx:
                return True
        return False
    
    def format_duration(self, seconds):
        """Format seconds as MM:SS."""
        minutes = seconds // 60
        seconds = seconds % 60
        return f"{minutes}:{seconds:02d}"
    
    def display_menu(self):
        """Display the interactive menu."""
        os.system('clear')
        print("Transcript Selector")
        if TERMIOS_AVAILABLE:
            print("Arrow keys: navigate | Space: start/end selection | Enter: finish | q: quit")
        else:
            print("w/s: navigate | space: start/end selection | enter: finish | q: quit")
        print(f"Total selected duration: {self.format_duration(self.get_total_selected_duration())}")
        print("-" * 80)
        
        # Show a window of lines around current position
        start_display = max(0, self.current_index - 10)
        end_display = min(len(self.lines), self.current_index + 11)
        
        for i in range(start_display, end_display):
            line = self.lines[i]
            prefix = ""
            
            # Current line indicator
            if i == self.current_index:
                prefix += ">"
            else:
                prefix += " "
            
            # Selection indicators
            if self.is_line_selected(i):
                prefix += "[X]"
            elif self.selecting and self.selection_start is not None and self.selection_start <= i <= self.current_index:
                prefix += "[?]"
            else:
                prefix += "   "
            
            # Truncate long text
            text = line['text']
            if len(text) > 60:
                text = text[:57] + "..."
            
            print(f"{prefix} {line['timestamp']} ({line['seconds']:3d}s) {text}")
        
        if self.selecting:
            current_range_duration = self.get_duration_for_range(self.selection_start, self.current_index)
            print(f"\nCurrent selection duration: {self.format_duration(current_range_duration)}")
    
    def get_key(self):
        """Get a single key press."""
        if TERMIOS_AVAILABLE:
            try:
                fd = sys.stdin.fileno()
                old_settings = termios.tcgetattr(fd)
                try:
                    tty.setcbreak(fd)
                    key = sys.stdin.read(1)
                    
                    # Handle arrow keys (escape sequences)
                    if key == '\x1b':
                        next_chars = sys.stdin.read(2)
                        key += next_chars
                        
                    return key
                finally:
                    termios.tcsetattr(fd, termios.TCSADRAIN, old_settings)
            except Exception as e:
                # If raw input fails, fall back to text input
                pass
        
        # Fallback method
        print("\nControls: w/s (up/down), space (select), enter (finish), q (quit)")
        key = input("Command: ").strip()
        if key.lower() == 'w':
            return '\x1b[A'  # Up arrow
        elif key.lower() == 's':
            return '\x1b[B'  # Down arrow
        elif key == ' ' or key.lower() == 'space':
            return ' '
        elif key.lower() == 'q':
            return 'q'
        elif key == '' or key.lower() == 'enter':
            return '\r'
        return key
    
    def run(self):
        """Run the interactive selector."""
        if not self.lines:
            print("No transcript data found.")
            return
        
        while True:
            self.display_menu()
            
            key = self.get_key()
            
            if key == 'q':
                break
            elif key == '\r' or key == '\n':  # Enter
                break
            elif key == ' ':  # Space - toggle selection
                if not self.selecting:
                    # Start new selection
                    self.selecting = True
                    self.selection_start = self.current_index
                else:
                    # End current selection
                    start_idx = min(self.selection_start, self.current_index)
                    end_idx = max(self.selection_start, self.current_index)
                    self.selected_ranges.append((start_idx, end_idx))
                    self.selecting = False
                    self.selection_start = None
            elif key == '\x1b[A':  # Up arrow
                if self.current_index > 0:
                    self.current_index -= 1
            elif key == '\x1b[B':  # Down arrow
                if self.current_index < len(self.lines) - 1:
                    self.current_index += 1
        
        # Print selected ranges
        if self.selected_ranges:
            print("\nSelected segments:")
            for i, (start_idx, end_idx) in enumerate(self.selected_ranges):
                start_line = self.lines[start_idx]
                end_line = self.lines[end_idx]
                duration = self.get_duration_for_range(start_idx, end_idx)
                print(f"Range {i+1}: {start_line['timestamp']} - {end_line['timestamp']} ({self.format_duration(duration)})")
            
            total_duration = self.get_total_selected_duration()
            print(f"\nTotal duration: {self.format_duration(total_duration)}")

def main():
    # Read from stdin or file
    if len(sys.argv) > 1:
        with open(sys.argv[1], 'r') as f:
            lines = f.readlines()
    else:
        lines = sys.stdin.readlines()
    
    selector = TranscriptSelector(lines)
    selector.run()

if __name__ == "__main__":
    main()