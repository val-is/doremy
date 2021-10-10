# doremy
Sleep tracking tool -- this is intended to be used in some experimentation that I'm doing.

## installation and running

to download:

```
git clone https://github.com/val-is/doremy.git
```

to configure:

copy `doremy/exampleconfig.json` to `doremy/config.json`, fill in the blanks

to run:

```
docker build -t doremy .
docker volume create doremy-vol
docker run -d \
    --name doremy \
    -v doremy-vol:/doremy \
    doremy
```

## misc

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

## a note on versioning and data storage
you know what's a really awesome design decision (/s)? not versioning data being stored.

so as it turns out, I wanted to change the way I collect data after a few months of doing it.
unfortunately, there's not a way to distinguish the two new datasets, aside from either

* a) physically storing them separately or
* b) just noting the time and splitting then when processing.

to deal with this in the future, the `additional-fields` field in the data storage json is now useful.
it copies over fields from `additional-fields` in the config file.
the default config specifies a `version` param that can be used to identify data being saved.
