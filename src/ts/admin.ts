
class Upload {
    private el: HTMLElement | null;
    private input: HTMLInputElement | null;

    constructor() {
        this.el = document.getElementById("import");
        this.input = null;
        if (this.el) {
            const input: HTMLInputElement | null = this.el.querySelector("input");
            if (input) {
                this.input = input;
            }
        }
        const b = document.getElementById("import-btn");
        if (b) {
            b.addEventListener("click", () => {
                this.click();
            }, false);
        }
    }

    public click(): void {
        if (this.el) {
            const input: HTMLInputElement | null = this.el.querySelector("input");
            if (input) {
                const files: FileList | null = input.files;
                if (files && files[0]) {
                    this.upload(files[0]);
                }
            }
        }
    }

    private upload(file: File): void {
        fetch("/admin/archive/import", {
            method: "POST",
            credentials: 'include',
            headers: new Headers({
                'Content-Type': ''
            }),
            body: file
        }).then(resp => {
            if (resp.ok) {
                this.input!.value = "";
            }
        }).catch(err => {
            console.error("error uploading", err);
        });
    }
}

class Deleter {
    constructor() {
        document.getElementById("delete-tweet")!.addEventListener("click", () => {
            this.click();
        }, false);
    }

    public click(): void {
        const el = document.getElementById("delete-tweet-id") as HTMLInputElement;
        fetch("/admin/delete?id=" + el!.value, {
            method: "GET",
            credentials: "include",
        }).then(() => {
            el!.value = "";
        }).catch(err => {
            console.error("error deleting tweet", err);
        });
    }
}

let upload = new Upload();
let deleter = new Deleter();
