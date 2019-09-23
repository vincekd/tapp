import { Component, Input } from "@angular/core";

import { AnalyticsService } from "../services/analytics.service";
import { Tweet } from "../interfaces/tweet";

@Component({
  selector: "[tweet-frag]",
  templateUrl: "/templates/tweet.html"
})
export class TweetFragComponent {
  // TODO: to constants?
  public likeIntentUrl: string = "https://twitter.com/intent/like?tweet_id=";
  public rtIntentUrl: string = "https://twitter.com/intent/retweet?tweet_id=";

  @Input("tweet")
  public tweet?: Tweet;
  @Input("showInternalLink")
  public showInternalLink?: boolean;

  constructor(private analServ: AnalyticsService) { }

  public trackClick(event: MouseEvent, which: string): void {
    this.analServ.trackEvent('click-external-link', which, 'button-' + event.button);
  }
}
