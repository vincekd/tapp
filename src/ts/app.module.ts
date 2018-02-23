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
import { UserService, TweetService } from './app.services'
import { CapitalizePipe, ReplaceMediaPipe } from './app.pipes'

const appRoutes: Routes = [
    { path : 'latest', component: LatestComponent },
    { path : 'best', component: BestComponent },
    { path : 'search', component: SearchComponent },
    { path : 'tweet/:id', component: TweetComponent },
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
        ReplaceMediaPipe
    ],
    providers: [UserService, TweetService],
    bootstrap: [TwitterAppComponent]
})
export class AppModule { }
