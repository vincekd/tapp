// angular imports
import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { HttpClientModule } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import {
    CommonModule
} from '@angular/common';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import {
    MatMenuModule,
    MatTooltipModule,
    MatSelectModule,
    MatSnackBarModule
} from '@angular/material';
import { RouterModule, Routes } from '@angular/router';

// library imports
import { InfiniteScrollModule } from 'ngx-infinite-scroll';

// project imports
import {
    TwitterAppComponent,
    MenuComponent,
    LatestComponent,
    BestComponent,
    SearchComponent,
    TweetComponent,
    ErrorPageComponent,
    TweetFragComponent,
    LoadingSpinnerComponent
} from './app.components';
import { UserService, TweetService, AnalyticsService } from './app.services';
import { CapitalizePipe, ReplaceMediaPipe, TweetDatePipe } from './app.pipes';

const appRoutes: Routes = [
    { path : 'latest', component: LatestComponent, data: { state: 'latest' } },
    { path : 'best', component: BestComponent, data: { state: 'best' } },
    { path : 'search', component: SearchComponent, data: { state: 'search' } },
    { path : 'tweet/:id', component: TweetComponent, data: { state: 'tweet-id' } },
    { path : 'error', component: ErrorPageComponent },
    { path : '', redirectTo: '/best', pathMatch: 'full' }
];

@NgModule({
    imports: [
        BrowserModule,
        RouterModule.forRoot(appRoutes, { enableTracing: false }),
        FormsModule,
        CommonModule,
        HttpClientModule,
        BrowserAnimationsModule,
        MatSelectModule,
        MatMenuModule,
        MatTooltipModule,
        MatSnackBarModule,
        InfiniteScrollModule
    ],
    declarations: [
        TwitterAppComponent,
        MenuComponent,
        LatestComponent,
        BestComponent,
        SearchComponent,
        TweetComponent,
        TweetFragComponent,
        LoadingSpinnerComponent,
        ErrorPageComponent,
        CapitalizePipe,
        ReplaceMediaPipe,
        TweetDatePipe
    ],
    providers: [UserService, TweetService, AnalyticsService],
    bootstrap: [TwitterAppComponent]
})
export class AppModule { }
