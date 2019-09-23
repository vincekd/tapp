import { Component, OnInit, } from '@angular/core';
import { Router, NavigationEnd, } from '@angular/router';

import { routerTransition } from '../app.animations';

import { UserService } from "../services/user.service";
import { AnalyticsService } from '../services/analytics.service';

import { User } from '../interfaces/user';

@Component({
  selector: 'twitter-app',
  animations: [routerTransition],
  template: `
<header ta-menu="" [user]="user"></header>
<div id="wrapper" [@routerTransition]="getState(o)">
  <router-outlet #o="outlet"></router-outlet>
</div>`,
})
export class TwitterAppComponent implements OnInit {
  public user?: User;

  constructor(
    private userServ: UserService,
    private analServ: AnalyticsService,
    private router: Router
  ) { }

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
