import torchaudio as ta
from chatterbox.tts import ChatterboxTTS
import sys
import glob
import random 

model = ChatterboxTTS.from_pretrained(device="mps")

text = sys.argv[1]
file = sys.argv[2]
print(text)
print(file)

# Select a random wav file from the voices directory
voice_files = glob.glob("/Users/aa/cs/voices/*.wav")
if not voice_files:
    raise FileNotFoundError("No wav files found in /Users/aa/cs/voices/")
AUDIO_PROMPT_PATH = random.choice(voice_files)
print(f"Using voice: {AUDIO_PROMPT_PATH}")
wav = model.generate(text, audio_prompt_path=AUDIO_PROMPT_PATH)
ta.save(file, wav, model.sr)
