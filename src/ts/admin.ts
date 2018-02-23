
// interface HTMLElement {
//     files: FileList;
// }

class Upload {
    el: HTMLElement | null;
    input: HTMLInputElement | null;
    constructor () {
        this.el = document.getElementById("import");
        this.input = null;
        if (this.el) {
            let input: HTMLInputElement | null = this.el.querySelector("input");
            if (input) {
                this.input = input;
            }
        }
        let b = document.getElementById("import-btn");
        if (b) {
            b.addEventListener("click", () => {
                this.click();
            }, false);
        }
    }

    click(): void {
        if (this.el) {
            let input: HTMLInputElement | null = this.el.querySelector("input");
            if (input) {
                let files: FileList | null = input.files;
                if (files && files[0]) {
                    // this.readFile(files[0]).then(r => {
                    //     this.upload(r);
                    // });
                    this.upload(files[0]);
                }
            }
        }
    }

    // private readFile(file: File): Promise<any> {
    //     return new Promise(resolve => {
    //         const reader = new FileReader();
    //         reader.onload = e => {
    //             resolve(e.currentTarget.result);
    //         };
    //         reader.readAs
    //     });
    // }

    private upload(file: File): void {
        //let fd = new FormData();
        // fd.append('name', "my-custom-upload");
        //fd.append('archive', file);

        fetch("/admin/archive/import", {
            method: "POST",
            credentials: 'include',
            headers: new Headers({
                //'Content-Type': 'multipart/form-data; charset=utf-8; boundary="-XXX"'
                'Content-Type': ''
            }),
            body: file
        }).then(resp => {
            console.log("uploaded!", resp);
            if (resp.ok) {
                //this.input!.value = "";
            }
        }).catch(err => {
            console.error("error uploading", err);
        });
    }
}

let upload = new Upload();
