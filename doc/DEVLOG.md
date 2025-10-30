# DevLog

Here I will keep track of the development progress of the project. This is just dump of what I think, do and planning to do after I have done checking out my work.

# Date: Sat Oct 25 23:11:19 +03 2025

I have just tidy up the project for better ergonomics moved things around. Added word and hint information to the game so I can show to the players. Next time I would like introduce to ability to guess the word in order to get points. My plan is to first finisht the basic mechanics of the user before adding any content like list of words, categories and so on. I will first make the game playable and enjoyable before introducing content.

I have learnt about `rm -f` usage for deleting files without error if the file does not exists. Also learnt about go templating extension and picked gohtml introduced a formatter for it and integrated in zed.


# Date: Tue Oct 28 22:17:47 +03 2025

I have introduced ability to guess the word and dynamic word masking by focusing on the data structure I need at the user interface rather than modeling from the database. From now on I am planning to focus first building the user and end result data structures fill them with current data as possible finally modify the database if there is no way to generate the state from the data. I will continue with End result focused for time being and in the future I will solve the optimization problems on the way when it is not managable.

# Fri Oct 31 01:32:43 +03 2025

I have added a lot this time. First of all I have added ability to save score value in database and laid the foundations of turn logic. While adding the feature, I have realized that I have too many bugs to follow on the repository layer of the application thus I have decided to add unit/integration tests for repository layer. On top of that I learnt that sqlite by default does not respect foreign keys and you have to explicitly tell to trigger related constraints on connection. I have started to feel that I might need to have a seperate controller and service layer which controller just takes input from requesgt and calls service with params for getting data to render the page/output. Still not going to introduce it because I want to see it replicated more.

One of the things I have learnt today is prepared statements and transactions in golang. Prepared statements are for optimizing which basically make DB layer skip preparing the QUERY so I didn't apply it yet! On the transactions side I have learnt couple of ways to integrate into the application which are introducing your own abstraction, abstracting at DB layer or just injecting directly to current DB layer. I chose to just use it with repository directly to have more explicit usage also this is another reason for having a seperate controller and service layer to not have both concerns at the same place thus reduce complexity for each method.
