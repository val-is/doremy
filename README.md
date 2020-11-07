# doremy
Sleep tracking tool -- this is intended to be used in some experimentation that I'm doing.

Specwise, I want this to be super easy to use as a Discord bot (for the time being).
Given that the use case of this is either a) when I'm super tired and about to fall asleep or b) when I've just woken up, there's not much room for scuffedness.

Usage:
- At night, wait for user to send something
- 7h later, post followup poll
- Wait for response and then store info
- Later, download stored data via endpoint for other app usage

In future:
- Generate data vis
- Control "experiments"
- Provide suggestions based on data trends

## configuration
copy `doremy/exampleconfig.json` to `doremy/config.json`, fill in the blanks