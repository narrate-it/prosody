`silence.mp3` is locally generated silence, not provider or copyrighted audio.
Generated only for tests with:

```
ffmpeg -nostdin -v error -f lavfi -i anullsrc=r=44100:cl=stereo -t 0.1 -c:a libmp3lame -b:a 128k -write_xing 0 -id3v2_version 0 -threads 1 silence.mp3
```

Production duration validation uses the Go decoder; ffmpeg is not a runtime dependency.
Encoder padding means the decoded duration exceeds 0.1 seconds.
