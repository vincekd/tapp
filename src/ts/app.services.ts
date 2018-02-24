import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';
import ('rxjs/add/operator/map');

import { User, Tweet } from './app.classes';

declare let ga: any;

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

    latestPage: number = 0;
    bestPage: number = 0;
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
        return this.get(which, this[which + "Page"].toString()).then(data => {
            if (data) {
                if (which === "latest") {
                    this.latestPage++;
                } else if (which === "best") {
                    this.bestPage++;
                }
                this[which].push(...data);
            }
            return data;
        });
    }

    searchTweets(search: string, page: string, order: string): Promise<Array<Tweet>> {
        return this.get("search", page, search, order);
    }

    get(which: string, page: string, search: string = '', order: string = '') {
        return this.http.get("/tweets", {
            params: {'type': which, 'page': page, 'search': search, 'order': order}
        }).toPromise().then(resp => {
            return <Array<Tweet>>resp;
        });
    }
}

@Injectable()
export class AnalyticsService {
    constructor() { }

    trackEvent(eventCategory: string, eventAction: string): void {
        //TODO: track likes/rts/twitter.com views
        if (typeof ga === "function") {
            try {
                ga('send', 'event', eventCategory, eventAction);
            } catch (e) {
                console.error("ga error", e);
            }
        } else {
            console.info("ga is undefined");
        }

    }

    trackPage(page: string): void {
        if (typeof ga === "function") {
            try {
                //ga('set', 'page', page);
                //ga('send', 'pageview');
                //ga('send', 'pageview', page, )
                ga('send', {
                    'hitType': 'pageview',
                    'page': page
                });
            } catch (e) {
                console.error("ga error", e);
            }
        } else {
            console.info("ga is undefined");
        }
    }
}
