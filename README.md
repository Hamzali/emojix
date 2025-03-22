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

