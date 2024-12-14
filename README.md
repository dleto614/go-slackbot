# go-slackbot

This was a proof of concept program that was for testing in my labs to see if using platforms like slack would make good c2 for stuff like red teaming. The idea originally came from articles where I read about threat actors using services sucha as slack, discord, telegram, etc as c2 communications.

It works for what I set out to do.

For configuration edit these lines:

```go
token := ""
channelID := ""
appToken := ""
TeamID = ""
```

This is for API and other information like team id and channel id needed by the API.

It isn't perfect, but I got it to work kind of.

APIs for some reason loveeeeeeeee to use IDs, but not always straightforward how to utilize them to get the functioanlity I wanted. In this case, sometimes I wanted to post in different channels, but the channels weren't always known. This was to kind of seperate different functions in the bot and also to seperate different computers running the bot so this way I could send commands to individual computers which I couldn't do without having the bot join different channels. (It is a weird solution, but I couldn't figure out how to get the bot to verify and send from the correct computer as is. This is probably working as designed and I am a dumbass.)
