# YouTube playlist video splitter

I made this programm for myself to download youtube videos that are playlists with music that seperated by chapters.

I use yt-dlp, jq, bash and ffmpeg. I download video, get all information i need using --dump-json argument (like title, timestamps for chapters and etc.) and then split this video into multiple mp3 files using ffmpeg.

It's not user frendly but it works (also it can only work on linux and you need bash as an available shell).
