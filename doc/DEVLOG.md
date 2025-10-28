# DevLog

Here I will keep track of the development progress of the project. This is just dump of what I think, do and planning to do after I have done checking out my work.

# Date: Sat Oct 25 23:11:19 +03 2025

I have just tidy up the project for better ergonomics moved things around. Added word and hint information to the game so I can show to the players. Next time I would like introduce to ability to guess the word in order to get points. My plan is to first finisht the basic mechanics of the user before adding any content like list of words, categories and so on. I will first make the game playable and enjoyable before introducing content.

I have learnt about `rm -f` usage for deleting files without error if the file does not exists. Also learnt about go templating extension and picked gohtml introduced a formatter for it and integrated in zed.


# Date: Tue Oct 28 22:17:47 +03 2025

I have introduced ability to guess the word and dynamic word masking by focusing on the data structure I need at the user interface rather than modeling from the database. From now on I am planning to focus first building the user and end result data structures fill them with current data as possible finally modify the database if there is no way to generate the state from the data. I will continue with End result focused for time being and in the future I will solve the optimization problems on the way when it is not managable.
