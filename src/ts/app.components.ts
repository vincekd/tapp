import {
    Component,
    Input,
    OnInit
} from '@angular/core';
import {
    ActivatedRoute,
    Router
} from '@angular/router';
import {
    MatSnackBar
} from "@angular/material";
import { User, Tweet } from './app.classes';
import { UserService, TweetService } from './app.services';

@Component({
    selector: 'twitter-app',
    template: `
<header ta-menu="" [user]="user"></header>
<div id="wrapper"><router-outlet></router-outlet></div>
`
})
export class TwitterAppComponent implements OnInit {
    user?: User;
    constructor(private userServ: UserService) { }
    ngOnInit(): void {
        console.info("TwitterAppComponent ngInit");
        this.userServ.getUser().subscribe(u => this.user = u);
    }
}

@Component({
    selector: "[ta-menu]",
    templateUrl: "/templates/menu.html"
})
export class MenuComponent {
    @Input() user?: User;
    constructor() { }
}

@Component({})
class TweetsComponent implements OnInit {
    protected tweetServ: TweetService;
    tweets?: Array<Tweet>;
    name: string = "best";
    scrollDistance: number = 2;
    scrollThrottle: number = 300;
    loading: boolean = false;

    constructor(tweetServ: TweetService) {
        this.tweetServ = tweetServ;
    }

    ngOnInit(): void {
        console.info("loading tweets:", this.name);
        this.loading = true;
        this.tweetServ.getTweets(this.name).then(tweets => {
            this.tweets = tweets;
            this.loading = false;
        }).catch(() => this.loading = false);
    }

    addTweets(): void {
        this.tweetServ.addTweets(this.name);
    }

    onScroll(): void {
        this.addTweets();
    }
}
@Component({
    templateUrl: "/templates/tweets.html"
})
export class LatestComponent extends TweetsComponent {
    constructor(tweetServ: TweetService) {
        super(tweetServ)
        this.name = "latest";
    }
}
@Component({
    templateUrl: "/templates/tweets.html"
})
export class BestComponent extends TweetsComponent {
    constructor(tweetServ: TweetService) {
        super(tweetServ)
        this.name = "best";
    }
}
@Component({
    templateUrl: "/templates/search.html"
})
export class SearchComponent extends TweetsComponent {
    search = {
        "text": "",
        "prev": {
            "text": "",
            "order": "",
            "ascending": false
        },
        "order": "Faves",
        "ascending": false,
        "resultsEmpty": false
    };
    sortOpts: Array<object> = [
        {label: "Created", name: "Id" },
        {label: "Best", name: "Faves"}
    ];
    page: number = 0;
    tweets: Array<Tweet> = [];

    constructor(tweetServ: TweetService, private snackBar: MatSnackBar, private router: Router, private activeRoute: ActivatedRoute) {
        super(tweetServ)
        this.name = "search";
    }

    ngOnInit(): void {
        this.activeRoute.queryParamMap.subscribe(params => {
            let split: any = {
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

    toggleSortOrder(): void {
        this.search.ascending = !this.search.ascending;
    }

    addTweets(): void {
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

    doSearch(): void {
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

    private getOrderStr(): string {
        return (this.search.ascending ? "" : "-") + this.search.order;
    }

    setLoc(): void {
        this.router.navigate(["/search"], {
            queryParams: {
                q: this.search.text,
                o: this.getOrderStr()
            }
        });
    }

    private checkChange(): boolean {
        return (this.search.text !== this.search.prev.text ||
                this.search.prev.order !== this.search.order ||
                this.search.prev.ascending !== this.search.ascending);
    }
}

@Component({
    template: `
<div id="single-tweet">
  <div></div>
  <div>
    <div class="tweet" *ngIf="tweet" tweet-frag="" [tweet]="tweet" [showInternalLink]="false"></div>
  </div>
  <div></div>
</div>
`
})
export class TweetComponent implements OnInit {
    //id?: string
    tweet?: Tweet
    constructor(private tweetServ: TweetService, private route: ActivatedRoute) { }

    ngOnInit(): void {
        this.route.params.subscribe(params => {
            let id: string = params['id'];
            if (id) {
                this.tweetServ.getTweet(id).then(tweet => {
                    this.tweet = tweet;
                });
            }
        });
    }
}

@Component({
    selector: "[tweet-frag]",
    templateUrl: "/templates/tweet.html"
})
export class TweetFragComponent {
    likeIntentUrl: string = "https://twitter.com/intent/like?tweet_id="
    rtIntentUrl: string = "https://twitter.com/intent/retweet?tweet_id="
    constructor() { }
    @Input("tweet") tweet?: Tweet;
    @Input("showInternalLink") showInternalLink?: boolean;
}

@Component({
    selector: "[loading-spinner]",
    template: `<div>
<div class="loading-spinner"><i class="icon-circle-notch"></i></div>
</div>`
})
export class LoadingSpinnerComponent {
    constructor() { }
}

@Component({
    templateUrl: "/templates/error.html"
})
export class ErrorPageComponent {
    constructor() { }
}
