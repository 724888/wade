Wade
====
The no-magic client-side web framework for Go (gopherjs)

  
Instructions on running the companion app Brogpal: https://github.com/phaikawl/brogpal.  

#How it works
##The flow
The server simply returns the `index.html` everytime the user visits the site, without any template/rendering on server side, the client controls the whole flow, render and direct the pages, the server is just a resource manager, an API provider, which returns needed resources (for example user info from the database) on ajax requests of the client, which is written in Go and compiled into Javascript. 

`index.html` is the root html, it imports `pages.html`, which defines the pages, and `elements.html`, which defines custom elements for the site.

##Pages
Each page is declared with a `wpage` element, the elements in the page are put inside of those tags. Pages need to be registered with `wade.Pager().RegisterPages()` in the client code. The Wade engine handles page switching (hide/show the appropriate elements). Wade uses HTML5 history to save page states and handle brower Forward/Back buttons.
Within the hierachy, pages can share elements with each other very easily without the need for any kind of inheritance or import.

##Custom elements
Custom elements can be declared with `welement` and registered with `wade.RegisterNewTag()`. This is useful for HTML code reuse.
Example:  
We define a custom element tag called "t-userinfo":

    <welement id="t-userinfo" attributes="Name Country">
        <p><strong>{Name}</strong>, <em>{Country}</em></p>
    </welement>

It's considered a html tag with attributes `Name` and `Age` now, we can use it like this:

    <t-userinfo attr-Name="Hai Thanh Nguyen" attr-Country="Vietnam">
Each custom element is bound to a specified model, which declares the datatype for attributes of that element. More on data binding below.

##Binding
Wade has support for data binding between HTML and Go/Js model.
For each page, a *page handler* could be registered with `wade.Pager().RegisterHandler` to control the page. Each page handler could return a model, it binds to the whole page.
For example:
If we have a struct

    type UserReg struct {
      Data struct {
        Username string
        Password string
      }
    }

The page handler:
    
    wade.Pager().RegisterHandler("pg-user-register", func() interface{} {
      return new(UserReg)
    })

The returned UserReg instance will be bound to the page, and for example within the page "pg-user-register", we have something like this:

    Username: <input type="text" bind-value="Data.Username"/>

The input field text will be bound/synchronized with the model's Data.Username field (specifically, if the user changes input, the model field changes, if the model field is updated, the input element is changed.

The binding mechanism is heavily influenced by [RivetsJs](http://rivetsjs.com), you could read some of its docs to gain more understanding.

#License

    Copyright (c) 2014, Hai Thanh Nguyen
    All rights reserved.

    Redistribution and use in source and binary forms, with or without
    modification, are permitted provided that the following conditions are met:
        * Redistributions of source code must retain the above copyright
          notice, this list of conditions and the following disclaimer.
        * Redistributions in binary form must reproduce the above copyright
          notice, this list of conditions and the following disclaimer in the
          documentation and/or other materials provided with the distribution.
        * Neither the name of VosDev nor the
          names of this software's contributors may be used to endorse or promote products
          derived from this software without specific prior written permission.

    THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
    ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
    WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
    DISCLAIMED. IN NO EVENT SHALL HAI THANH NGUYEN BE LIABLE FOR ANY
    DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
    (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
    LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
    ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
    (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
    SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
