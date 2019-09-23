import { Component } from "@angular/core";

import { TweetService } from "../services/tweet.service";
import { TweetsComponent } from "./tweets.component";

@Component({
  templateUrl: "/templates/tweets.html"
})
export class BestComponent extends TweetsComponent {
  constructor(tweetServ: TweetService) {
    super(tweetServ);
    this.name = "best";
  }
}
