import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: "Capitalize"
})
export class CapitalizePipe implements PipeTransform {
  public transform(str: string): string {
    return str.substring(0, 1).toUpperCase() + str.substring(1);
  }
}
