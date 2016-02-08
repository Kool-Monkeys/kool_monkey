import {AUTH0_CLIENT_ID} from './auth0-variables'
import {AUTH0_DOMAIN} from './auth0-variables'

export var lock = new Auth0Lock(AUTH0_CLIENT_ID, AUTH0_DOMAIN)

export default {
  checkAuth () {
    if (localStorage.getItem('id_token')) {
      return true
    } else {
      return false
    }
  }
}
