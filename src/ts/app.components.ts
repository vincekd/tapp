import {
    Component,
    Input,
    OnInit,
} from '@angular/core';
import {
    ActivatedRoute,
    Router,
    NavigationEnd
} from '@angular/router';
import {
    MatSnackBar
} from "@angular/material";
import { User, Tweet } from './app.classes';
import { UserService, TweetService, AnalyticsService } from './app.services';
import {
    routerTransition
} from './app.animations';

@Component({
    selector: 'twitter-app',
    animations: [routerTransition],
    template: `
<header ta-menu="" [user]="user"></header>
<div id="wrapper" [@routerTransition]="getState(o)">
    <router-outlet #o="outlet"></router-outlet>
</div>
`,
})
export class TwitterAppComponent implements OnInit {
    public user?: User;
    constructor(private userServ: UserService, private analServ: AnalyticsService, private router: Router) { }
    public ngOnInit(): void {
        console.info("TwitterAppComponent ngInit");
        this.userServ.getUser().then(u => this.user = u);
        this.router.events.subscribe((e: any) => {
            if (e instanceof NavigationEnd) {
                if (e.url) {
                    this.analServ.trackPage(e.url);
                }
            }
        });
    }
    public getState(outlet): any {
        return outlet.activatedRouteData.state;
    }
}

@Component({
    selector: "[ta-menu]",
    templateUrl: "/templates/menu.html"
})
export class MenuComponent {
    @Input()
    public user?: User;
    //constructor() { }
}

class TweetsComponent implements OnInit {
    public tweets?: Tweet[];
    public name: string = "best";
    public scrollDistance: number = 2;
    public scrollThrottle: number = 300;
    public loading: boolean = false;

    protected tweetServ: TweetService;

    constructor(tweetServ: TweetService) {
        this.tweetServ = tweetServ;
    }

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

@Component({
    templateUrl: "/templates/tweets.html"
})
export class LatestComponent extends TweetsComponent {
    constructor(tweetServ: TweetService) {
        super(tweetServ);
        this.name = "latest";
    }
}

@Component({
    templateUrl: "/templates/tweets.html"
})
export class BestComponent extends TweetsComponent {
    constructor(tweetServ: TweetService) {
        super(tweetServ);
        this.name = "best";
    }
}

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

@Component({
    selector: "[tweet-frag]",
    templateUrl: "/templates/tweet.html"
})
export class TweetFragComponent {
    public likeIntentUrl: string = "https://twitter.com/intent/like?tweet_id=";
    public rtIntentUrl: string = "https://twitter.com/intent/retweet?tweet_id=";
    public favstarUrl: string = "https://favstar.fm/users/" + '@' + "/status/";

    @Input("tweet")
    public tweet?: Tweet;
    @Input("showInternalLink")
    public showInternalLink?: boolean;

    constructor(private analServ: AnalyticsService, userServ: UserService) {
        userServ.getUser().then(u => {
            this.favstarUrl = "https://favstar.fm/users/" + u.ScreenName + "/status/";
        });
    }

    public trackClick(event: MouseEvent, which: string): void {
        this.analServ.trackEvent('click-external-link', which, 'button-' + event.button);
    }
}

@Component({
    selector: "[loading-spinner]",
    template: `<div>
  <div class="loading-spinner">
    <svg version="1.1" id="loader-1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px"
         width="40px" height="40px" viewBox="0 0 40 40" enable-background="new 0 0 40 40" xml:space="preserve">
      <path opacity="0.2" fill="#000" d="M20.201,5.169c-8.254,0-14.946,6.692-14.946,14.946c0,8.255,6.692,14.946,14.946,14.946 s14.946-6.691,14.946-14.946C35.146,11.861,28.455,5.169,20.201,5.169z M20.201,31.749c-6.425,0-11.634-5.208-11.634-11.634 c0-6.425,5.209-11.634,11.634-11.634c6.425,0,11.633,5.209,11.633,11.634C31.834,26.541,26.626,31.749,20.201,31.749z"/>
      <path fill="#000" d="M26.013,10.047l1.654-2.866c-2.198-1.272-4.743-2.012-7.466-2.012h0v3.312h0 C22.32,8.481,24.301,9.057,26.013,10.047z">
        <animateTransform attributeType="xml"
            attributeName="transform"
            type="rotate"
            from="0 20 20"
            to="360 20 20"
            dur="0.7s"
            repeatCount="indefinite"/>
      </path>
    </svg>
  </div>
</div>`,
})
export class LoadingSpinnerComponent {
    //constructor() { }
}

@Component({
    templateUrl: "/templates/error.html",
})
export class ErrorPageComponent {
    //constructor() { }
}
