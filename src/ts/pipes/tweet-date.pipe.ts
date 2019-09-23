import { Pipe, PipeTransform } from '@angular/core';
import { DatePipe } from '@angular/common';

import { Tweet } from "../interfaces/tweet";

@Pipe({
  name: "TweetDate"
})
export class TweetDatePipe implements PipeTransform {
  private datePipe: DatePipe;
  constructor() {
    this.datePipe = new DatePipe(this.getLang());
  }

  public transform(tweet: Tweet): string {
    return this.datePipe.transform(new Date(tweet.Created * 1000), 'MMM d, y, h:mm a') || '';
  }

  private getLang(): string {
    if (navigator.languages && navigator.languages.length > 0) {
      return navigator.languages[0];
    }
    return navigator.language;
  }
}
