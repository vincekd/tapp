import { Component } from "@angular/core";
import { MatSnackBar } from "@angular/material";
import { Router, ActivatedRoute, } from '@angular/router';

import { TweetService } from "../services/tweet.service";
import { TweetsComponent } from "./tweets.component";

import { Tweet } from "../interfaces/tweet";

@Component({
  templateUrl: "/templates/search.html"
})
export class SearchComponent extends TweetsComponent {
  public search = {
    text: "",
    prev: {
      text: "",
      order: "",
      ascending: false
    },
    order: "Faves",
    ascending: false,
    resultsEmpty: false
  };
  public sortOpts: object[] = [
    {label: "Created", name: "Id" },
    {label: "Best", name: "Faves"}
  ];
  public page: number = 0;
  public tweets: Tweet[] = [];

  constructor(tweetServ: TweetService, private snackBar: MatSnackBar,
              private router: Router, private activeRoute: ActivatedRoute) {
    super(tweetServ);
    this.name = "search";
  }

  public ngOnInit(): void {
    this.activeRoute.queryParamMap.subscribe(params => {
      const split: any = {
        q: decodeURIComponent(params.get("q") || ""),
        o: params.get("o")
      };
      this.search.text = split.q || "";
      if (split.o) {
        if (split.o.startsWith("-")) {
          this.search.ascending = false;
          this.search.order = split.o.substring(1);
        } else {
          this.search.order = split.o;
        }
      }
      if (this.search.text.length > 0) {
        this.doSearch();
      }
    });
  }

  public toggleSortOrder(): void {
    this.search.ascending = !this.search.ascending;
  }

  public addTweets(): void {
    this.tweetServ.searchTweets(this.search.text, this.page.toString(), this.getOrderStr()).then(tweets => {
      if (tweets) {
        this.tweets.push(...tweets);
      } else {
        this.search.resultsEmpty = true;
      }
      this.page++;
      this.loading = false;
    }).catch(() => {
      this.loading = false;
    }); //finally not supported
  }

  public doSearch(): void {
    if (this.search.text.length <= 2) {
      this.snackBar.open("Search is too short.", "Dismiss", {duration: 5000});
    } else if (!this.checkChange()) {
      this.snackBar.open("Search parameters haven't changed.", "Dismiss", {duration: 5000});
    } else {
      this.page = 0;
      this.tweets = [];

      this.search.prev.text = this.search.text;
      this.search.prev.order = this.search.order;
      this.search.prev.ascending = this.search.ascending;

      this.search.resultsEmpty = false;

      this.loading = true;

      this.addTweets();
    }
  }

  public setLoc(): void {
    this.router.navigate(["/search"], {
      queryParams: {
        q: this.search.text,
        o: this.getOrderStr()
      }
    });
  }

  private getOrderStr(): string {
    return (this.search.ascending ? "" : "-") + this.search.order;
  }

  private checkChange(): boolean {
    return (this.search.text !== this.search.prev.text ||
            this.search.prev.order !== this.search.order ||
            this.search.prev.ascending !== this.search.ascending);
  }
}
