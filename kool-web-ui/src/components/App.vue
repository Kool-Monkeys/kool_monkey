<template>
  <div id="wrapper">
    <nav class="navbar navbar-default">
      <div class="container">
        <ul class="nav navbar-nav">
          <li><a v-link="'home'">Home</a></li>
          <li><a v-link="'dashboard'" v-show="authenticated">Dashboard</a></li>
          <li><a v-link="'settings'" v-show="authenticated">Settings</a></li>
          <li><a v-link="'#'" @click="login()" v-show="!authenticated">Login</a>
              <a v-link="'#'" @click="logout()" v-show="authenticated">Logout</a></li>
        </ul>
      </div>    
    </nav>
    <div class="container">
      <router-view></router-view>
    </div>
  </div>
</template>

<script>
  import auth from '../auth'
  import {lock} from '../auth'

  export default {
    ready() {
      this.authenticated = auth.checkAuth();
    },
    data() {
      return {
        authenticated: false,
        secretThing: ''
      }
    },
    methods: {
      login() {
        var self = this;
        
        lock.show((err, profile, token) => {
          if(err) {          
            // Handle the error
            console.log(err)          
          } else {
            // Set the token and user profile in local storage
            localStorage.setItem('profile', JSON.stringify(profile));
            localStorage.setItem('id_token', token);
            self.authenticated = true;
          }        
        });
      },
      logout() {
        var self = this;
        // To log out, we just need to remove the token and profile
        // from local storage
        localStorage.removeItem('id_token');
        localStorage.removeItem('profile');
        self.authenticated = false;      
      },
      // Make a secure call to the server by attaching
      // the user's JWT as an Authorization header
      getSecretThing() {
        var jwtHeader = { 'Authorization': 'Bearer ' + localStorage.getItem('id_token') };
        this.$http.get('http://localhost:3001/secured/ping', (data) => {
          console.log(data)
          this.secretThing = data.text;
        }, { 
          headers: jwtHeader
        }).error((err) => console.log(err));
      }
    }
  }
</script>
