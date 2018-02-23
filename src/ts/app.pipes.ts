import { Pipe, PipeTransform } from '@angular/core';
import { Tweet } from "./app.classes";
//import { DatePipe } from '@angular/common';
//import { DateFormatter } from '@angular/src'

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
