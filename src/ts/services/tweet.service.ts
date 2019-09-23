import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import { Tweet } from '../interfaces/tweet';

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
