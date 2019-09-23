import { OnInit, } from '@angular/core';

import { TweetService } from "../services/tweet.service";

import { Tweet } from '../interfaces/tweet';

export class TweetsComponent implements OnInit {
  public tweets?: Tweet[];
  public name: string = "best";
  public scrollDistance: number = 2;
  public scrollThrottle: number = 300;
  public loading: boolean = false;

  constructor(
    protected tweetServ: TweetService
  ) { }

  public ngOnInit(): void {
    console.info("loading tweets:", this.name);
    this.loading = true;
    this.tweetServ.getTweets(this.name).then(tweets => {
      this.tweets = tweets;
      this.loading = false;
    }).catch(() => this.loading = false);
  }

  public addTweets(): void {
    this.tweetServ.addTweets(this.name);
  }

  public onScroll(): void {
    this.addTweets();
  }
}
