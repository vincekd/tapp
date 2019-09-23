import { Component, OnInit } from "@angular/core";
import { ActivatedRoute } from "@angular/router";

import { TweetService } from "../services/tweet.service";
import { Tweet } from "../interfaces/tweet";

@Component({
  template: `
<div id="single-tweet">
  <div></div>
  <div>
    <div class="tweet" *ngIf="tweet" tweet-frag="" [tweet]="tweet" [showInternalLink]="false"></div>
  </div>
  <div></div>
</div>`
})
export class TweetComponent implements OnInit {
  //id?: string
  public tweet?: Tweet;
  constructor(private tweetServ: TweetService, private route: ActivatedRoute) { }

  public ngOnInit(): void {
    this.route.params.subscribe(params => {
      const id: string = params.id;
      if (id) {
        this.tweetServ.getTweet(id).then(tweet => {
          this.tweet = tweet;
        });
      }
    });
  }
}
