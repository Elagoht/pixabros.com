interface RequestLogin {
  username: string;
  password: string;
}

interface ResponseLogin {
  username: string;
}

interface ResponseWhoami {
  username: string;
}

interface RequestChangePassword {
  current_password: string;
  new_password: string;
}
