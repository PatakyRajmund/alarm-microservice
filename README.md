# Alarm Microservice
**The project includes a microservice written in Golang that handles the registration, authentication of new users and stores active user data in a database. It is meant to work together with Home Assistant as a custom authentication extension for an alarm.**

## Abstract Visualization of how it works:
<img width="581" height="261" alt="asd" src="https://github.com/user-attachments/assets/a5e0cbe3-dfb3-41fa-ae8e-b0c136c79b9a" />

In the microservice itself the strict order of the authentication (User -> Authentication Subsystem -> Microservice) and the fact that the only request source of the QR code MUST be Home Assistant is **not** regulated, it only gives you API that can help easily achieve this architecture. It also does not implement the functionality of the authentication subsystem (which is quite easy to implement). And the Automations triggered in HA are also not implemented (to change the state of your alarm). 

## What it does:

- It stores user + password + ttl records in a database and keeps the database consistent (with a goroutine that runs in every hour and removes expired/invalid records) AND if a record expires in the meantime it is still excluded in checking when an authentication request happens.
- Stores the password securely (with hashing + salting).
- Keeps track of "logged in" users (if a user sends an authentication request when already authenticated it gets "logged out").
- It triggers Home Assistant Webhooks when a user gets home (if the house was previously empty) or if the last "logged in" user logs out. The Webhooks MUST be different.
- Gives an API for the user to manipulate the data in the DB. 

## API Endpoints
It gives the user an API for manipulating the database:
- [PUT] **/api/adduser/{user}/{ttl}** : For adding a new record for {user} into the database. The password is "random" generated (it is a v4 UUID), so on its own quite difficult to guess. And with {ttl} one can set an expiration time period for the record (in hours)
-  [DELETE] **/api/delete/{user}** : For removing the record of {user} from the database.
- [POST] **/api/remove-invalid-records** : For the explicit removal of expired records.

It gives an API for the authentication subsystem (for instance a QR Code Reader on the same PC):
- [GET] **/api/authenticate/{user}?password={password}** : For authenticating {user} with {password}. The whole URL (including this URI) is the content of the QR code. _When used it SHOULD be set up to only accept authentication requests from the subsystem._

It gives Home Assistant an API for getting the QR Code:
-  [GET] **/api/getcode/{user}** : For retrieving the QR Code that has the URL to call for the authentication of {user}. **MUST BE DEFENDED**

