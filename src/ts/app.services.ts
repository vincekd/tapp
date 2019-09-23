import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import { User, Tweet } from './app.classes';

declare let ga: any;

@Injectable()
export class UserService {
    private user: Promise<User>;
    constructor(private http: HttpClient) {
        this.user = this.get();
    }

    public getUser(): Promise<User> {
        return this.user;
    }

    private get(): Promise<User> {
        return this.http.get("/user").toPromise().then(res => res as User);
    }

}

@Injectable()
export class TweetService {
    public latest: Tweet[] = [];
    public best: Tweet[] = [];
    private latestPage: number = 0;
    private bestPage: number = 0;

    constructor(private http: HttpClient) { }

    public getTweet(id: string): Promise<Tweet> {
        return this.http.get("/tweet", {
            params: {id}
        }).toPromise().then(resp => {
            return resp as Tweet;
        });
    }

    public getTweets(which: string): Promise<Tweet[]> {
        if (this[which].length === 0) {
            return this.addTweets(which).then(() => {
                return this[which];
            });
        }
        return Promise.all([this[which]]).then(d => d[0]);
    }

    public addTweets(which: string): Promise<Tweet[]> {
        const page: number = (which === "latest" ? this.latestPage : this.bestPage);
        return this.get(which, page.toString()).then(data => {
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

    public searchTweets(search: string, page: string, order: string): Promise<Tweet[]> {
        return this.get("search", page, search, order);
    }

    public get(which: string, page: string, search: string = '', order: string = '') {
        const params: any = {page};
        if (which === "search") {
            params.search = search;
            params.order = order;
        }
        return this.http.get("/tweets/" + which, {params}).
            toPromise().then(resp => resp as Tweet[]);
    }
}

@Injectable()
export class AnalyticsService {
    private enabled: boolean = false;

    constructor() {
      this.enabled = (typeof ga === "function");
      if (!this.enabled) {
            console.warn("ganalytics not loaded.");
        }
    }

    public trackEvent(eventCategory: string, event: string, button: string): void {
        if (this.enabled) {
            try {
                ga('send', 'event', eventCategory, event, button);
            } catch (e) {
                console.error("ga error", e);
            }
        }
    }

    public trackPage(page: string): void {
        if (this.enabled) {
            try {
                ga('send', 'pageview', page);
            } catch (e) {
                console.error("ga error", e);
            }
        }
    }
}
