# Game Room Flows

- click on new game and it will create a new game room on the fly and redirect you to the new game page
- enter game room id and click join and it will redirect to the game room page
- directly go to game room link
- a user can only be in one game room at a time.
- if a user is in a game room and went to the index page it will redirect user to the joined game
- if a user is already in a game room and clicked another link, it will remove from current and join to clicked game room
- when user leaves a game room it will redirect to the main page
- rooms will be shareable via links


# Game Room Data

list of players

scoreboard - score of each user

messages - text content typed by users with timestamps

events - correct guesses, joins, leaves,


# AI generated guess items

It is really time consuming to build a turn system at the first stage because it requires timers, realtime chat, UX elements and input validations for emoji interactions. In order to create the first playable version users will not decide on any items to guess or emojis to represents. There will be a list of movies with emoji representation and N number of hints with revealing levels. This list can be generated with AI and also can be extended and corrected by hand. There will be no turn mechanism but there will be timer for given hints. Game logic will be as follows.


## room logic
MIN_PLAYER - number of people required to play the game
MAX_PLAYER - number of maximum player count

If there is at least MIN_PLAYER number of players start a COUNTDOWN and start the game if COUNTDOWN finishes
- what if a user leaves before COUNTDOWN is not completed; reset the COUNTDOWN and wait for MIN_PLAYER number of player

If there are MAX_PLAYER number of people then just start the game

## game session
- There will be a list of movies to go through
- Game starts with the first item and go thorughs all items in the movie list spending X amount of time
- If everybody guesses correct skips to next item without waiting for timer to expire.
- At end of the list every player's score is calculated and build a scoreboard.

## round details


## score calcuation
parameters
- TIME_LEFT: 1 - 0,how much percentage of time left since round start
- ORDER: 0 - N, order which the user guessed first, second or Nth
- HINT_REVEALED: H - 0, how many hints left



# Strategy

First implement related to game room user flows without other in game actions like messaging, score etc.
Then have game rooms with data and state but without actions figure out how the game room will look with data.

# EMOJIX: Charades with emojis

target of this project is building an realtime applicaiton using almost no dependency. Try to be DUMB! Encourage the GRUG brain mode. Also I would like to hone my golang skills while learning VIM motions.
About infrastructure, I would like to make it simple as possible. It will be deployed into a linux machine as a monolith. No horizontal scaling only vertical. If I can I will try to test the limits of the capacity of my realtime server.

## rulest of development
- only one main file
- no third party dependencies

## technology stack

## deployment

## rules of the game:
- at least 2 player required
- Every turn a player tries to tell a word using an emoji keyboard in 2 minutes and others try to guess it.
- Everybody collects points for telling and guessing.
- At end user with the most points wins.

## point system:
GUESSER
	- +5 points for guessing right.
	- +1 point for every seconds left
	- -1 point for every wrong guess

TELLER
	- +1 point if someone guesses right
	- +1 point for every seconds left
	- +5 point if everyon guesses right

## user stories
ANON PLAYER
	[ ] as a player, I would like to join a game using a link
	[ ] as a player, I would like to start a game
	[ ] as a plyaer, I would liket to share a game link
	[ ] as a player, I would like to signup with email, nickname and password

IN GAME
	[ ] as a player, I would like to see the time left
	[ ] as a player, I would like to see the score board
	[ ] as a player, I would like to see if I am teller or guesser
	[ ] as a player, I would like to see the current teller

GUESSER
	[ ] as a player, I would like to ask questions to teller
	[ ] as a player, I would like to guess the word being told

TELLER
	[ ] as a player, I would like to send hints with emojis
