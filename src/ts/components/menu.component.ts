import { Component, Input, } from '@angular/core';

import { User } from '../interfaces/user';

@Component({
  selector: "[ta-menu]",
  templateUrl: "/templates/menu.html"
})
export class MenuComponent {
  @Input()
  public user?: User;
}
