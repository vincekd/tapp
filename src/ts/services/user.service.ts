import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

import { User } from '../interfaces/user';

@Injectable()
export class UserService {
  private user: Promise<User>;
  constructor(private http: HttpClient) {
    this.user = this.get();
  }

  public getUser(): Promise<User> {
    return this.user;
  }

  private get(): Promise<User> {
    return this.http.get("/user").toPromise().then(res => res as User);
  }
}

