
export interface User {
    ScreenName: string,
	Id: number,
	Url: string,
	ProfileImageUrlHttps: string,
	Name: string,
	Description: string,
	Followers: number,
	Following: number
	TweetCount: number,
	Location: string,
	Verified: boolean,
	Link: string,
	Updated: number,
}

export interface Tweet {
    Ratio: number,
	IdStr: string,
	Faves: number,
	Rts: number,
	Id: number,
	Created: number,
	Updated: number,
	Text: string,
    Url: string,
    Media: Array<Media>
}

export interface Media {
    IdStr: string,
	Url: string,
	ExpandedUrl: string,
	Type: string,
	MediaUrl: string,
}
