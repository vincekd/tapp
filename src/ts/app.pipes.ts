import { Pipe, PipeTransform } from '@angular/core';
import { Tweet } from "./app.classes";
import { DatePipe } from '@angular/common';

@Pipe({
    "name": "Capitalize"
})
export class CapitalizePipe implements PipeTransform {
    transform(str: string): string {
        return str.substring(0, 1).toUpperCase() + str.substring(1);
    }
}

@Pipe({
    "name": "ReplaceMedia"
})
export class ReplaceMediaPipe implements PipeTransform {
    transform(tweet: Tweet): string {
        if (tweet.Media) {
            let text = tweet.Text;
            tweet.Media.forEach(m => {
                text = text.replace(m.Url, `<img src="${m.MediaUrl}" />`)
            });
            return text;
        }
        return tweet.Text;
    }
}

@Pipe({
    "name": "TweetDate"
})
export class TweetDatePipe implements PipeTransform {
    private datePipe: DatePipe
    constructor() {
        this.datePipe = new DatePipe(this.getLang())
    }

    transform(tweet: Tweet): string {
        return this.datePipe.transform(new Date(tweet.Created*1000), 'MMM d, y, h:mm a') || '';
    }

    private getLang(): string {
        if (navigator.languages && navigator.languages.length > 0) {
            return navigator.languages[0];
        }
        return navigator.language;
    }
}
