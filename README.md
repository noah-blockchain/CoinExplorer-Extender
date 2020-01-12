<p align="center" style="text-align: center;">
    <a href="https://github.com/noah-blockchain/noah-extender/blob/master/LICENSE">
        <img src="https://img.shields.io/packagist/l/doctrine/orm.svg" alt="License">
    </a>
    <img alt="undefined" src="https://img.shields.io/github/last-commit/noah-blockchain/noah-extender.svg">
</p>

# Noah Explorer Extender

The official repository of Noah Explorer Extender service.

Extender is a service responsible for seeding the database from the blockchain network. Part of the Noah Explorer service.

_NOTE: This project in active development stage so feel free to send us questions, issues, and wishes_

## BUILD

- make create_vendor
- make build

## Configure Extender Service from Environment (example in .env.example)
1) Set up connect to PostgresSQL Databases.
2) Set up connect to Node which working in non-validator mode. 
3) Set up connect to Extender service. 

## RUN
If you run Extender for the first time yoг need to run  [Explorer Genesis Uploader](https://github.com/noah-blockchain/explorer-genesis-uploader)
to fill data from genesis file (you can use the same config file for both services)

./extender

_We recommend use our official docker image._
### Important Environments
Example for all important environments you can see in file .env.example.
Its config for connect to PostgresSQL, Node API URL, Extender URL and service mode (debug, prod).

