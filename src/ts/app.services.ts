import { BigNumber } from 'bignumber.js';
import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
//import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/map';
import { User, Tweet } from './app.classes';


@Injectable()
export class UserService {
    constructor(private http: HttpClient) {
        this.user = this.get();
    }
    private user: Observable<User>;

    get(): Observable<User> {
        return this.http.get("/user").map(res => <User>res);
    }
    getUser(): Observable<User> {
        return this.user;
    }
}

@Injectable()
export class TweetService {
    constructor(private http: HttpClient) { }

    latestLastId: string = "";
    bestLastId: string = "";
    bestPage: number = 1;
    latest: Array<Tweet> = [];
    best: Array<Tweet> = [];

    getTweet(id: string): Promise<Tweet> {
        return this.http.get("/tweet", {
            params: {"id": id}
        }).toPromise().then(resp => {
            return <Tweet>resp;
        });
    }

    getTweets(which: string): Promise<Array<Tweet>> {
        if (this[which].length === 0) {
            return this.addTweets(which).then(() => {
                return this[which];
            });
        }
        return Promise.all([this[which]]).then(d => d[0]);
    }

    addTweets(which: string): Promise<Array<Tweet>> {
        return this.get(which, this.latestLastId, this.bestPage.toString(), '', '').then(data => {
            if (data) {
                if (which === "latest") {
                    this.latestLastId = data.reduce((min, tweet) => {
                        if (!min || new BigNumber(tweet.IdStr).isLessThan(min)) {
                            return tweet.IdStr;
                        }
                        return min;
                    }, this.latestLastId);
                } else if (which === "best") {
                    this.bestPage++;
                }
                this[which].push(...data);
            }
            return data;
        });
    }

    searchTweets(search: string, page: string, order: string): Promise<Array<Tweet>> {
        return this.get("search", "", page, search, order);
    }

    get(which: string, lastId: string, page: string, search: string, order: string) {
        return this.http.get("/tweets", {
            params: {'type': which, 'lastId': lastId, 'page': page, 'search': search, 'order': order}
        }).toPromise().then(resp => {
            return <Array<Tweet>>resp;
        });
    }
}
