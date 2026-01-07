# TODOs

## Wed Jan  7 12:47:13 AM +03 2026

game
- [x] introduce guessed event for realtime update of guess word if the user guessed (remove the mask)
- [x] realtime for turn end when everyone guessed the word
- [x] when a user is kicked and flagged as inactive, have a logic to rejoin the user, reuse the record and mutate back to active if there is space in the game

techdebt
- [] find a way to have one way to fetch leaderboard info, currently code is duplicated for Leaderboard and GameState

## Backlog

plan
- introduce proper designs to the app
- game should end when all of the words are guessed in a list of words, what to show after game ends, have a proper word lists
- improve join experience, a user should be able to join a game just by navigating to the game url

game(next)
- [] show more letter as hints if guess has matching letters like wordle
- [] timer for ending the turn, no need to guess everyone to end the turn
- [] add player state for disconnected

user
- [] nickaneme edit (should I save the user data?)
- [] user sign up with email
- [] user email verification

content
- [] word lists
- [] word list editor for signed up users

techdebt
- [] cleanup anon users in the database? Search how people do it and implement a pragmatic simple solution
- [] fetch all game data with one sql query in one go (get game data/state)
- [] cleanup/close realtime channels for users properly or atleast figure out if they are cleaned up by go runtime
- [] add e2e tests (when you nail down the initial version)


## initial todos
game
- [x] add emoji to guess when game starts
- [x] check for guess logic on message send
- [x] hide or show movie name and reflect on user for guess state
- [x] active/inactive player concept
- [x] room capacity depending on the active user not all user registered in.
- [x] realtime leave game logic on inactivity
- [x] realtime leave event
- [x] realtime join game event

infra
- [x] realtime with sse
- [x] introduce htmx for client side reactivity (working on this)

techdebt
- [x] add tests for usecase layer
