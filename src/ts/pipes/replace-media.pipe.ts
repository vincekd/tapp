import { Pipe, PipeTransform } from '@angular/core';

import { Tweet } from "../interfaces/tweet";

@Pipe({
  name: "ReplaceMedia"
})
export class ReplaceMediaPipe implements PipeTransform {
  public transform(tweet: Tweet): string {
    if (tweet.Media) {
      let text = tweet.Text;
      tweet.Media.forEach(m => {
        const url = "/media?file=" + m.UploadFileName;
        text = text.replace(m.Url, `<img src="${url}" />`);
      });
      return text;
    }
    return tweet.Text;
  }
}
